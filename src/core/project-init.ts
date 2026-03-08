import fs from "node:fs/promises";
import path from "node:path";
import {
    BASE_WORKSPACE_RELATIVE_PATH,
    CONFIG_RELATIVE_PATH,
    DEFAULT_LOGS_RELATIVE_DIR,
    PROJECT_TOOLKIT_DIRNAME,
} from "./config.js";

export interface ProjectInitResult {
  created: string[];
  updated: string[];
  skipped: string[];
}

export async function initializeProjectToolkit(cwd: string, force: boolean): Promise<ProjectInitResult> {
  const toolkitDir = path.join(cwd, PROJECT_TOOLKIT_DIRNAME);
  const configPath = path.join(cwd, CONFIG_RELATIVE_PATH);
  const baseWorkspacePath = path.join(cwd, BASE_WORKSPACE_RELATIVE_PATH);
  const projectName = path.basename(cwd);

  await fs.mkdir(toolkitDir, { recursive: true });

  const result: ProjectInitResult = {
    created: [],
    updated: [],
    skipped: [],
  };

  await writeScaffoldFile({
    filePath: configPath,
    content: buildConfigTemplate(projectName),
    force,
    result,
    cwd,
  });

  await writeScaffoldFile({
    filePath: baseWorkspacePath,
    content: buildBaseWorkspaceTemplate(),
    force,
    result,
    cwd,
  });

  return result;
}

async function writeScaffoldFile(options: {
  filePath: string;
  content: string;
  force: boolean;
  result: ProjectInitResult;
  cwd: string;
}): Promise<void> {
  const { filePath, content, force, result, cwd } = options;
  const relativePath = path.relative(cwd, filePath) || path.basename(filePath);

  const exists = await fileExists(filePath);
  if (exists && !force) {
    result.skipped.push(relativePath);
    return;
  }

  await fs.writeFile(filePath, content, "utf8");

  if (exists) {
    result.updated.push(relativePath);
    return;
  }

  result.created.push(relativePath);
}

async function fileExists(filePath: string): Promise<boolean> {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

function buildConfigTemplate(projectName: string): string {
  return `project:
  name: ${projectName}

dev:
  # command: npm run dev

logs:
  dir: ${DEFAULT_LOGS_RELATIVE_DIR}

workspace:
  baseFile: ${BASE_WORKSPACE_RELATIVE_PATH}

shared:
  # Share gitignored files from the main repo into generated worktrees.
  # source/target default to path when omitted.
  # include/exclude match worktree names.
  - path: .env
`;
}

function buildBaseWorkspaceTemplate(): string {
  return `${JSON.stringify(
    {
      folders: [],
      settings: {},
    },
    null,
    2,
  )}
`;
}
