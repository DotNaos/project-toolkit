import { execFile } from "node:child_process";
import fs from "node:fs/promises";
import path from "node:path";
import { promisify } from "node:util";
import type { RepoContext, RepoContextFilePreview } from "./types.js";

const execFileAsync = promisify(execFile);
const KEY_FILE_CANDIDATES = [
  "README.md",
  "package.json",
  "tsconfig.json",
  "go.mod",
  "Makefile",
  "Dockerfile",
  ".project-toolkit/config.yaml",
];

export async function collectRepoContext(cwd: string): Promise<RepoContext> {
  const [gitRoot, gitBranch, topLevelEntries, filePreviews] = await Promise.all([
    readGitValue(cwd, ["rev-parse", "--show-toplevel"]),
    readGitValue(cwd, ["branch", "--show-current"]),
    readTopLevelEntries(cwd),
    readFilePreviews(cwd),
  ]);

  return {
    cwd,
    gitRoot,
    gitBranch,
    topLevelEntries,
    filePreviews,
  };
}

export function renderRepoContext(context: RepoContext): string {
  const sections: string[] = [
    `Current working directory: ${context.cwd}`,
    `Git root: ${context.gitRoot ?? "not detected"}`,
    `Git branch: ${context.gitBranch ?? "not detected"}`,
    `Top-level entries: ${context.topLevelEntries.join(", ") || "(none)"}`,
  ];

  if (context.filePreviews.length > 0) {
    sections.push(
      "Key file previews:\n" +
        context.filePreviews
          .map((preview) => `- ${preview.path}\n${indent(preview.preview, "  ")}`)
          .join("\n"),
    );
  }

  return sections.join("\n\n");
}

async function readTopLevelEntries(cwd: string): Promise<string[]> {
  const entries = await fs.readdir(cwd, { withFileTypes: true });
  return entries
    .filter((entry) => entry.name !== ".git" && entry.name !== "node_modules")
    .map((entry) => entry.name + (entry.isDirectory() ? "/" : ""))
    .sort((left, right) => left.localeCompare(right))
    .slice(0, 40);
}

async function readFilePreviews(cwd: string): Promise<RepoContextFilePreview[]> {
  const previews = await Promise.all(
    KEY_FILE_CANDIDATES.map(async (relativePath) => {
      const absolutePath = path.join(cwd, relativePath);
      try {
        const stat = await fs.stat(absolutePath);
        if (!stat.isFile()) {
          return null;
        }

        const source = await fs.readFile(absolutePath, "utf8");
        return {
          path: relativePath,
          preview: createPreview(source),
        } satisfies RepoContextFilePreview;
      } catch {
        return null;
      }
    }),
  );

  return previews.filter((value): value is RepoContextFilePreview => value !== null);
}

async function readGitValue(cwd: string, args: string[]): Promise<string | null> {
  try {
    const { stdout } = await execFileAsync("git", args, { cwd });
    const value = stdout.trim();
    return value || null;
  } catch {
    return null;
  }
}

function createPreview(source: string): string {
  const lines = source.split(/\r?\n/).slice(0, 20).join("\n").trim();
  return lines.length > 1200 ? `${lines.slice(0, 1200)}...` : lines;
}

function indent(source: string, prefix: string): string {
  return source
    .split("\n")
    .map((line) => `${prefix}${line}`)
    .join("\n");
}
