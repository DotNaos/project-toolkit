import { execFile } from "node:child_process";
import fs from "node:fs/promises";
import { promisify } from "node:util";
import { ProjectToolkitError } from "./errors.js";
import { deriveProjectKey, getManagedWorktreePath } from "./project-paths.js";
import { generateProjectWorkspace, type WorkspaceGenerateResult } from "./workspace.js";
import type { ProjectToolkitConfig } from "./types.js";

const execFileAsync = promisify(execFile);

export interface CreateProjectWorktreeOptions {
  cwd: string;
  config: ProjectToolkitConfig;
  worktreeName: string;
  branchName?: string;
  baseRef?: string;
  workspaceName?: string;
  workspaceOutputPath?: string;
}

export interface CreateProjectWorktreeResult {
  branchName: string;
  gitRoot: string;
  worktreeName: string;
  worktreePath: string;
  workspace: WorkspaceGenerateResult;
}

export async function createProjectWorktree(options: CreateProjectWorktreeOptions): Promise<CreateProjectWorktreeResult> {
  const worktreeName = normalizeName(options.worktreeName, "worktree name");
  const branchName = normalizeName(options.branchName ?? worktreeName, "branch name");
  const workspaceName = normalizeName(options.workspaceName ?? worktreeName, "workspace name");
  const gitRoot = await resolveGitRoot(options.cwd);
  const projectKey = deriveProjectKey(options.cwd, options.config);
  const worktreePath = getManagedWorktreePath(projectKey, worktreeName);

  await ensurePathAvailable(worktreePath, "worktree path");

  const branchExists = await gitBranchExists(gitRoot, branchName);
  const addWorktreeOptions: Parameters<typeof addGitWorktree>[0] = {
    gitRoot,
    worktreePath,
    branchName,
    branchExists,
  };

  if (options.baseRef) {
    addWorktreeOptions.baseRef = options.baseRef;
  }

  await addGitWorktree(addWorktreeOptions);

  try {
    const workspaceOptions: Parameters<typeof generateProjectWorkspace>[0] = {
      cwd: options.cwd,
      config: options.config,
      workspaceName,
      targetRoot: worktreePath,
    };

    if (options.workspaceOutputPath) {
      workspaceOptions.outputPath = options.workspaceOutputPath;
    }

    const workspace = await generateProjectWorkspace(workspaceOptions);

    return {
      branchName,
      gitRoot,
      worktreeName,
      worktreePath,
      workspace,
    };
  } catch (error) {
    await removeGitWorktree(gitRoot, worktreePath);
    throw error;
  }
}

async function resolveGitRoot(cwd: string): Promise<string> {
  const gitRoot = await runGit(cwd, ["rev-parse", "--show-toplevel"]);
  const normalized = gitRoot.trim();
  if (!normalized) {
    throw new ProjectToolkitError("Current working directory must be inside a Git repository");
  }

  return normalized;
}

async function ensurePathAvailable(targetPath: string, label: string): Promise<void> {
  try {
    await fs.access(targetPath);
    throw new ProjectToolkitError(`${label} already exists: ${targetPath}`);
  } catch (error) {
    if (isNodeError(error) && error.code === "ENOENT") {
      return;
    }

    throw error;
  }
}

async function gitBranchExists(cwd: string, branchName: string): Promise<boolean> {
  try {
    await execFileAsync("git", ["show-ref", "--verify", "--quiet", `refs/heads/${branchName}`], { cwd });
    return true;
  } catch {
    return false;
  }
}

async function addGitWorktree(options: {
  gitRoot: string;
  worktreePath: string;
  branchName: string;
  branchExists: boolean;
  baseRef?: string;
}): Promise<void> {
  const args = ["worktree", "add"];
  if (!options.branchExists) {
    args.push("-b", options.branchName);
  }

  args.push(options.worktreePath);

  if (options.branchExists) {
    args.push(options.branchName);
  } else if (options.baseRef) {
    args.push(options.baseRef);
  }

  await runGit(options.gitRoot, args);
}

async function removeGitWorktree(gitRoot: string, worktreePath: string): Promise<void> {
  try {
    await runGit(gitRoot, ["worktree", "remove", "--force", worktreePath]);
  } catch {
    // Best effort cleanup only.
  }
}

async function runGit(cwd: string, args: string[]): Promise<string> {
  try {
    const { stdout } = await execFileAsync("git", args, { cwd });
    return stdout.trim();
  } catch (error) {
    throw new ProjectToolkitError(`Git command failed: git ${args.join(" ")} (${getErrorMessage(error)})`);
  }
}

function normalizeName(value: string, label: string): string {
  const normalized = value.trim();
  if (!normalized) {
    throw new ProjectToolkitError(`${label} must be a non-empty string`);
  }

  return normalized;
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