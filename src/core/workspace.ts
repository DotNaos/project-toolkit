import { ParseErrorCode, parse as parseJsonc, printParseErrorCode } from "jsonc-parser";
import fs from "node:fs/promises";
import path from "node:path";
import { BASE_WORKSPACE_RELATIVE_PATH } from "./config.js";
import { ProjectToolkitError } from "./errors.js";
import { deriveProjectKey, getGeneratedWorkspacePath } from "./project-paths.js";
import type { ProjectToolkitConfig, ProjectToolkitSharedLink } from "./types.js";

export interface WorkspaceGenerateOptions {
  cwd: string;
  config: ProjectToolkitConfig;
  workspaceName: string;
  outputPath?: string;
  targetRoot?: string;
}

export interface WorkspaceSharedLinkResult {
  path: string;
  sourcePath: string;
  targetPath: string;
  status: "linked" | "skipped" | "conflict" | "missing-source";
  reason?: string;
}

export interface WorkspaceGenerateResult {
  workspaceName: string;
  projectKey: string;
  baseWorkspacePath: string;
  outputPath: string;
  targetRoot: string;
  folderPath: string;
  sharedLinks: WorkspaceSharedLinkResult[];
}

export async function generateProjectWorkspace(options: WorkspaceGenerateOptions): Promise<WorkspaceGenerateResult> {
  const workspaceName = normalizeWorkspaceName(options.workspaceName);
  const projectKey = deriveProjectKey(options.cwd, options.config);
  const targetRoot = path.resolve(options.cwd, options.targetRoot ?? options.cwd);
  const baseWorkspacePath = resolveBaseWorkspacePath(options.cwd, options.config);
  const outputPathOptions: Parameters<typeof resolveOutputPath>[0] = {
    cwd: options.cwd,
    projectKey,
    workspaceName,
  };

  if (options.outputPath) {
    outputPathOptions.outputPath = options.outputPath;
  }

  const outputPath = resolveOutputPath(outputPathOptions);

  await ensureDirectory(targetRoot, "workspace root");

  const baseWorkspace = await readBaseWorkspace(baseWorkspacePath);
  const workspaceDir = path.dirname(outputPath);
  const folderPath = toPortablePath(path.relative(workspaceDir, targetRoot) || ".");

  const generatedWorkspace = {
    ...baseWorkspace,
    folders: [{ path: folderPath }],
  };

  await fs.mkdir(workspaceDir, { recursive: true });
  await fs.writeFile(outputPath, `${JSON.stringify(generatedWorkspace, null, 2)}\n`, "utf8");

  const sharedLinks = await applySharedLinks({
    sourceRoot: options.cwd,
    targetRoot,
    workspaceName,
    sharedLinks: options.config.shared ?? [],
  });

  return {
    workspaceName,
    projectKey,
    baseWorkspacePath,
    outputPath,
    targetRoot,
    folderPath,
    sharedLinks,
  };
}

function normalizeWorkspaceName(value: string): string {
  const normalized = value.trim();
  if (!normalized) {
    throw new ProjectToolkitError("Workspace name must be a non-empty string");
  }

  return normalized;
}

function resolveBaseWorkspacePath(cwd: string, config: ProjectToolkitConfig): string {
  const configuredPath = config.workspace?.baseFile ?? BASE_WORKSPACE_RELATIVE_PATH;
  return path.isAbsolute(configuredPath) ? configuredPath : path.resolve(cwd, configuredPath);
}

function resolveOutputPath(options: {
  cwd: string;
  outputPath?: string;
  projectKey: string;
  workspaceName: string;
}): string {
  if (options.outputPath) {
    return path.isAbsolute(options.outputPath)
      ? options.outputPath
      : path.resolve(options.cwd, options.outputPath);
  }

  return getGeneratedWorkspacePath(options.projectKey, options.workspaceName);
}

async function readBaseWorkspace(filePath: string): Promise<Record<string, unknown>> {
  let source: string;
  try {
    source = await fs.readFile(filePath, "utf8");
  } catch (error) {
    throw new ProjectToolkitError(`Failed to read workspace base file ${filePath}: ${getErrorMessage(error)}`);
  }

  const parseErrors: { error: ParseErrorCode; offset: number; length: number }[] = [];
  const parsed = parseJsonc(source, parseErrors, {
    allowTrailingComma: true,
    disallowComments: false,
  });

  if (parseErrors.length > 0) {
    const first = parseErrors[0];
    if (!first) {
      throw new ProjectToolkitError(`Invalid workspace base file ${filePath}`);
    }

    throw new ProjectToolkitError(
      `Invalid workspace base file ${filePath}: ${printParseErrorCode(first.error)} at offset ${first.offset}`,
    );
  }

  if (!isRecord(parsed)) {
    throw new ProjectToolkitError(`Workspace base file ${filePath} must contain a JSON object`);
  }

  return parsed;
}

async function applySharedLinks(options: {
  sourceRoot: string;
  targetRoot: string;
  workspaceName: string;
  sharedLinks: ProjectToolkitSharedLink[];
}): Promise<WorkspaceSharedLinkResult[]> {
  const results: WorkspaceSharedLinkResult[] = [];

  for (const entry of options.sharedLinks) {
    results.push(await applySharedLink(entry, options));
  }

  return results;
}

async function applySharedLink(
  entry: ProjectToolkitSharedLink,
  options: {
    sourceRoot: string;
    targetRoot: string;
    workspaceName: string;
  },
): Promise<WorkspaceSharedLinkResult> {
  const resolvedPaths = resolveSharedLinkPaths(entry, options.sourceRoot, options.targetRoot);

  if (!matchesWorkspace(entry, options.workspaceName)) {
    return buildSharedLinkResult(entry.path, resolvedPaths, "skipped", `workspace '${options.workspaceName}' is filtered out`);
  }

  if (resolvedPaths.sourcePath === resolvedPaths.targetPath) {
    return buildSharedLinkResult(entry.path, resolvedPaths, "skipped", "source and target are identical");
  }

  const sourceStats = await safeLstat(resolvedPaths.sourcePath);
  if (!sourceStats) {
    return buildSharedLinkResult(entry.path, resolvedPaths, "missing-source", "source path does not exist");
  }

  await fs.mkdir(path.dirname(resolvedPaths.targetPath), { recursive: true });

  const existingTargetState = await inspectExistingTarget(resolvedPaths.targetPath, resolvedPaths.sourcePath);
  if (existingTargetState) {
    return buildSharedLinkResult(entry.path, resolvedPaths, existingTargetState.status, existingTargetState.reason);
  }

  const relativeSourcePath = path.relative(path.dirname(resolvedPaths.targetPath), resolvedPaths.sourcePath) || ".";
  await fs.symlink(relativeSourcePath, resolvedPaths.targetPath, sourceStats.isDirectory() ? "dir" : "file");
  return buildSharedLinkResult(entry.path, resolvedPaths, "linked");
}

function resolveSharedLinkPaths(entry: ProjectToolkitSharedLink, sourceRoot: string, targetRoot: string): {
  sourcePath: string;
  targetPath: string;
} {
  return {
    sourcePath: path.resolve(sourceRoot, entry.source ?? entry.path),
    targetPath: path.resolve(targetRoot, entry.target ?? entry.path),
  };
}

async function inspectExistingTarget(
  targetPath: string,
  sourcePath: string,
): Promise<{ status: "skipped" | "conflict"; reason: string } | null> {
  const targetStats = await safeLstat(targetPath);
  if (!targetStats) {
    return null;
  }

  const alreadyLinked = await isSymlinkToTarget(targetPath, sourcePath);
  return {
    status: alreadyLinked ? "skipped" : "conflict",
    reason: alreadyLinked ? "shared link already exists" : "target path already exists",
  };
}

function buildSharedLinkResult(
  sharedPath: string,
  resolvedPaths: { sourcePath: string; targetPath: string },
  status: WorkspaceSharedLinkResult["status"],
  reason?: string,
): WorkspaceSharedLinkResult {
  const result: WorkspaceSharedLinkResult = {
    path: sharedPath,
    sourcePath: resolvedPaths.sourcePath,
    targetPath: resolvedPaths.targetPath,
    status,
  };

  if (reason) {
    result.reason = reason;
  }

  return result;
}

function matchesWorkspace(entry: ProjectToolkitSharedLink, workspaceName: string): boolean {
  if (entry.include?.includes(workspaceName) === false) {
    return false;
  }

  if (entry.exclude?.includes(workspaceName)) {
    return false;
  }

  return true;
}

async function ensureDirectory(dirPath: string, label: string): Promise<void> {
  const stats = await safeLstat(dirPath);
  if (!stats) {
    throw new ProjectToolkitError(`${label} does not exist: ${dirPath}`);
  }

  if (!stats.isDirectory()) {
    throw new ProjectToolkitError(`${label} must be a directory: ${dirPath}`);
  }
}

async function safeLstat(filePath: string): Promise<Awaited<ReturnType<typeof fs.lstat>> | null> {
  try {
    return await fs.lstat(filePath);
  } catch (error) {
    if (isNodeError(error) && error.code === "ENOENT") {
      return null;
    }

    throw error;
  }
}

async function isSymlinkToTarget(linkPath: string, sourcePath: string): Promise<boolean> {
  const linkStats = await fs.lstat(linkPath);
  if (!linkStats.isSymbolicLink()) {
    return false;
  }

  const [resolvedLinkPath, resolvedSourcePath] = await Promise.all([
    fs.realpath(linkPath),
    fs.realpath(sourcePath),
  ]);

  return resolvedLinkPath === resolvedSourcePath;
}

function toPortablePath(value: string): string {
  return value.split(path.sep).join("/");
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isNodeError(error: unknown): error is NodeJS.ErrnoException {
  return error instanceof Error && "code" in error;
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  try {
    return JSON.stringify(error);
  } catch {
    return "unknown error";
  }
}
