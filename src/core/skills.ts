import fs from "node:fs/promises";
import path from "node:path";
import { parse as parseYaml } from "yaml";
import { ProjectToolkitError } from "./errors.js";
import type { LoadedSkill, SkillSummary } from "./types.js";

interface ParsedMarkdown {
  metadata: Record<string, unknown>;
  body: string;
}

const FRONT_MATTER_PATTERN = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/;

export async function discoverSkills(skillsRoot: string): Promise<SkillSummary[]> {
  const entries = await fs.readdir(skillsRoot, { withFileTypes: true });
  const skills = await Promise.all(
    entries
      .filter((entry) => entry.isDirectory())
      .map((entry) => inspectSkill(path.join(skillsRoot, entry.name), entry.name)),
  );

  return skills.sort((left, right) => left.id.localeCompare(right.id));
}

export async function loadSkill(skillsRoot: string, skillId: string): Promise<LoadedSkill> {
  const rootDir = path.join(skillsRoot, skillId);

  try {
    await fs.access(rootDir);
  } catch {
    throw new ProjectToolkitError(`Unknown skill: ${skillId}`);
  }

  const skill = await inspectSkill(rootDir, skillId);
  if (!skill.valid) {
    const detail = skill.errors.length > 0 ? ` (${skill.errors.join("; ")})` : "";
    throw new ProjectToolkitError(`Skill "${skillId}" is not runnable${detail}`);
  }

  return skill;
}

async function inspectSkill(rootDir: string, skillId: string): Promise<LoadedSkill> {
  const normalizedMetadataPath = path.join(rootDir, "skill.yaml");
  const normalizedPromptPath = path.join(rootDir, "prompt.md");
  const legacySkillPath = path.join(rootDir, "SKILL.md");

  if (await pathExists(normalizedMetadataPath) || await pathExists(normalizedPromptPath)) {
    return inspectNormalizedSkill(rootDir, skillId, normalizedMetadataPath, normalizedPromptPath);
  }

  if (await pathExists(legacySkillPath)) {
    return inspectLegacySkill(rootDir, skillId, legacySkillPath);
  }

  return inspectMarkdownFallback(rootDir, skillId);
}

async function inspectNormalizedSkill(
  rootDir: string,
  skillId: string,
  metadataPath: string,
  promptPath: string,
): Promise<LoadedSkill> {
  const errors: string[] = [];

  let metadata: Record<string, unknown> = {};
  if (await pathExists(metadataPath)) {
    metadata = await readYamlObject(metadataPath, errors);
  } else {
    errors.push("missing skill.yaml");
  }

  let prompt = "";
  if (await pathExists(promptPath)) {
    prompt = (await fs.readFile(promptPath, "utf8")).trim();
    if (!prompt) {
      errors.push("prompt.md is empty");
    }
  } else {
    errors.push("missing prompt.md");
  }

  const title = readString(metadata, "title") ?? readString(metadata, "name");
  const description = readString(metadata, "description");
  const promptPathValue = await pathExists(promptPath) ? promptPath : null;
  const metadataPathValue = await pathExists(metadataPath) ? metadataPath : null;

  return {
    id: skillId,
    title: title ?? skillId,
    prompt,
    valid: errors.length === 0,
    format: errors.length === 0 ? "normalized" : "invalid",
    rootDir,
    errors,
    ...(description ? { description } : {}),
    ...(promptPathValue ? { promptPath: promptPathValue } : {}),
    ...(metadataPathValue ? { metadataPath: metadataPathValue } : {}),
  };
}

async function inspectLegacySkill(rootDir: string, skillId: string, skillPath: string): Promise<LoadedSkill> {
  const markdown = await fs.readFile(skillPath, "utf8");
  const parsed = parseMarkdown(markdown);
  const title =
    readString(parsed.metadata, "title") ??
    readString(parsed.metadata, "name") ??
    extractHeading(parsed.body) ??
    skillId;
  const description = readString(parsed.metadata, "description");
  const prompt = parsed.body.trim() || buildMetadataPrompt(title, description);
  const errors = prompt ? [] : ["SKILL.md does not contain usable prompt content"];

  return {
    id: skillId,
    title,
    prompt,
    valid: errors.length === 0,
    format: errors.length === 0 ? "legacy" : "invalid",
    rootDir,
    errors,
    promptPath: skillPath,
    ...(description ? { description } : {}),
  };
}

async function inspectMarkdownFallback(rootDir: string, skillId: string): Promise<LoadedSkill> {
  const entries = await fs.readdir(rootDir, { withFileTypes: true });
  const markdownFiles = entries
    .filter((entry) => entry.isFile() && entry.name.toLowerCase().endsWith(".md"))
    .map((entry) => entry.name)
    .sort((left, right) => left.localeCompare(right));

  if (markdownFiles.length !== 1) {
    return {
      id: skillId,
      title: skillId,
      prompt: "",
      valid: false,
      format: "invalid",
      rootDir,
      errors: ["missing skill definition (expected skill.yaml + prompt.md, SKILL.md, or a single markdown file)"],
    };
  }

  const [markdownFile] = markdownFiles;
  if (!markdownFile) {
    throw new ProjectToolkitError(`Unable to resolve markdown fallback for ${skillId}`);
  }

  const promptPath = path.join(rootDir, markdownFile);
  const promptSource = await fs.readFile(promptPath, "utf8");
  const parsed = parseMarkdown(promptSource);
  const title = extractHeading(parsed.body) ?? skillId;
  const prompt = parsed.body.trim() || promptSource.trim();
  const errors = prompt ? [] : ["markdown skill file is empty"];

  return {
    id: skillId,
    title,
    prompt,
    valid: errors.length === 0,
    format: errors.length === 0 ? "markdown-fallback" : "invalid",
    rootDir,
    errors,
    promptPath,
  };
}

async function pathExists(targetPath: string): Promise<boolean> {
  try {
    await fs.access(targetPath);
    return true;
  } catch {
    return false;
  }
}

async function readYamlObject(sourcePath: string, errors: string[]): Promise<Record<string, unknown>> {
  try {
    const raw = await fs.readFile(sourcePath, "utf8");
    const parsed = parseYaml(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    errors.push(`${path.basename(sourcePath)} must contain a YAML object`);
    return {};
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown error";
    errors.push(`failed to parse ${path.basename(sourcePath)}: ${message}`);
    return {};
  }
}

function parseMarkdown(source: string): ParsedMarkdown {
  const match = source.match(FRONT_MATTER_PATTERN);
  if (!match) {
    return { metadata: {}, body: source };
  }

  try {
    const frontMatter = match[1];
    if (!frontMatter) {
      return { metadata: {}, body: source };
    }

    const metadata = parseYaml(frontMatter);
    return {
      metadata: metadata && typeof metadata === "object" && !Array.isArray(metadata)
        ? (metadata as Record<string, unknown>)
        : {},
      body: source.slice(match[0].length),
    };
  } catch {
    return { metadata: {}, body: source };
  }
}

function buildMetadataPrompt(title: string, description?: string): string {
  return [title, description].filter(Boolean).join("\n\n").trim();
}

function extractHeading(source: string): string | null {
  const match = source.match(/^#\s+(.+)$/m);
  return match?.[1]?.trim() ?? null;
}

function readString(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}
