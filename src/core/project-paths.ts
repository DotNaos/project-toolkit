import { createHash } from "node:crypto";
import os from "node:os";
import path from "node:path";
import { PROJECT_TOOLKIT_DIRNAME } from "./config.js";
import type { ProjectToolkitConfig } from "./types.js";

export function deriveProjectKey(cwd: string, config: ProjectToolkitConfig): string {
  const baseName = slugify(config.project?.name ?? path.basename(cwd));
  const hash = createHash("sha1").update(cwd).digest("hex").slice(0, 8);
  return `${baseName}-${hash}`;
}

export function getProjectStateRoot(projectKey: string): string {
  return path.join(os.homedir(), PROJECT_TOOLKIT_DIRNAME, "projects", projectKey);
}

export function getGeneratedWorkspacePath(projectKey: string, workspaceName: string): string {
  return path.join(getProjectStateRoot(projectKey), "workspaces", `${workspaceName}.code-workspace`);
}

export function getManagedWorktreePath(projectKey: string, worktreeName: string): string {
  return path.join(getProjectStateRoot(projectKey), "worktrees", worktreeName);
}

function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replaceAll(/[^a-z0-9]+/g, "-").replaceAll(/^-+|-+$/g, "");
  return normalized || "project";
}