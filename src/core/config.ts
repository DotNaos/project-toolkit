import fs from "node:fs/promises";
import path from "node:path";
import { parse as parseYaml } from "yaml";
import { ProjectToolkitError } from "./errors.js";
import type { ProjectToolkitConfig, ProjectToolkitDevRouterConfig, ProjectToolkitSharedLink } from "./types.js";

export const PROJECT_TOOLKIT_DIRNAME = ".project-toolkit";
export const CONFIG_RELATIVE_PATH = path.join(PROJECT_TOOLKIT_DIRNAME, "config.yaml");
export const BASE_WORKSPACE_RELATIVE_PATH = path.join(PROJECT_TOOLKIT_DIRNAME, "base.code-workspace");
export const DEFAULT_LOGS_RELATIVE_DIR = path.join("logs", "project-toolkit");

export async function loadProjectToolkitConfig(cwd: string): Promise<ProjectToolkitConfig> {
  const configPath = path.join(cwd, CONFIG_RELATIVE_PATH);

  let source: string;
  try {
    source = await fs.readFile(configPath, "utf8");
  } catch (error) {
    if (isNodeError(error) && error.code === "ENOENT") {
      return {};
    }

    throw new ProjectToolkitError(`Failed to read ${CONFIG_RELATIVE_PATH}: ${getErrorMessage(error)}`);
  }

  let parsed: unknown;
  try {
    parsed = parseYaml(source);
  } catch (error) {
    throw new ProjectToolkitError(`Invalid ${CONFIG_RELATIVE_PATH}: ${getErrorMessage(error)}`);
  }

  if (parsed == null) {
    return {};
  }

  if (!isRecord(parsed)) {
    throw new ProjectToolkitError(`${CONFIG_RELATIVE_PATH} must contain a YAML object`);
  }

  return validateConfig(parsed, CONFIG_RELATIVE_PATH);
}

function validateConfig(raw: Record<string, unknown>, label: string): ProjectToolkitConfig {
  const dev = readNestedObject(raw, "dev", label);
  const logs = readNestedObject(raw, "logs", label);
  const project = readNestedObject(raw, "project", label);
  const workspace = readNestedObject(raw, "workspace", label);

  const devArgs = readOptionalStringArray(dev ?? {}, "args", `${label}.dev.args`);
  const devCommand = readOptionalString(dev, "command", `${label}.dev.command`);
  const devRouter = readDevRouter(dev, `${label}.dev`);
  const logsDir = readOptionalString(logs, "dir", `${label}.logs.dir`);
  const projectName = readOptionalString(project, "name", `${label}.project.name`);
  const workspaceBaseFile = readOptionalString(workspace, "baseFile", `${label}.workspace.baseFile`);
  const shared = readSharedLinks(raw, label);

  const config: ProjectToolkitConfig = {};

  if (devArgs || devCommand || devRouter) {
    if (devRouter && !devArgs && !devCommand) {
      throw new ProjectToolkitError(`${label}.dev.router requires ${label}.dev.command or ${label}.dev.args`);
    }

    config.dev = {};
    if (devArgs) {
      config.dev.args = devArgs;
    }

    if (devCommand) {
      config.dev.command = devCommand;
    }

    if (devRouter) {
      config.dev.router = devRouter;
    }
  }

  if (logsDir) {
    config.logs = { dir: logsDir };
  }

  if (projectName) {
    config.project = { name: projectName };
  }

  if (workspaceBaseFile) {
    config.workspace = { baseFile: workspaceBaseFile };
  }

  if (shared.length > 0) {
    config.shared = shared;
  }

  return config;
}

function readDevRouter(
  raw: Record<string, unknown> | undefined,
  label: string,
): ProjectToolkitDevRouterConfig | undefined {
  const value = readNestedObject(raw ?? {}, "router", label);
  if (!value) {
    return undefined;
  }

  const mode = readRequiredString(value, "mode", `${label}.router.mode`);
  if (mode !== "portless" && mode !== "dockportless") {
    throw new ProjectToolkitError(`${label}.router.mode must be 'portless' or 'dockportless'`);
  }

  const name = readOptionalString(value, "name", `${label}.router.name`);
  const router: ProjectToolkitDevRouterConfig = { mode };
  if (name) {
    router.name = name;
  }

  return router;
}

function readSharedLinks(raw: Record<string, unknown>, label: string): ProjectToolkitSharedLink[] {
  const value = raw.shared;
  if (value === undefined || value === null) {
    return [];
  }

  if (!Array.isArray(value)) {
    throw new ProjectToolkitError(`${label}.shared must be an array`);
  }

  return value.map((entry, index) => validateSharedLink(entry, `${label}.shared[${index}]`));
}

function validateSharedLink(value: unknown, label: string): ProjectToolkitSharedLink {
  if (!isRecord(value)) {
    throw new ProjectToolkitError(`${label} must be an object`);
  }

  const sharedPath = readRequiredString(value, "path", `${label}.path`);
  const source = readOptionalString(value, "source", `${label}.source`);
  const target = readOptionalString(value, "target", `${label}.target`);
  const include = readOptionalStringArray(value, "include", `${label}.include`);
  const exclude = readOptionalStringArray(value, "exclude", `${label}.exclude`);

  const sharedLink: ProjectToolkitSharedLink = {
    path: sharedPath,
  };

  if (source) {
    sharedLink.source = source;
  }

  if (target) {
    sharedLink.target = target;
  }

  if (include) {
    sharedLink.include = include;
  }

  if (exclude) {
    sharedLink.exclude = exclude;
  }

  return sharedLink;
}

function readNestedObject(
  raw: Record<string, unknown>,
  key: string,
  label: string,
): Record<string, unknown> | undefined {
  const value = raw[key];
  if (value === undefined || value === null) {
    return undefined;
  }

  if (!isRecord(value)) {
    throw new ProjectToolkitError(`${label}.${key} must be an object`);
  }

  return value;
}

function readOptionalString(
  raw: Record<string, unknown> | undefined,
  key: string,
  label: string,
): string | undefined {
  if (!raw || !(key in raw)) {
    return undefined;
  }

  const value = raw[key];
  if (typeof value !== "string") {
    throw new ProjectToolkitError(`${label} must be a string`);
  }

  const normalized = value.trim();
  return normalized.length > 0 ? normalized : undefined;
}

function readRequiredString(raw: Record<string, unknown>, key: string, label: string): string {
  const value = readOptionalString(raw, key, label);
  if (!value) {
    throw new ProjectToolkitError(`${label} must be a non-empty string`);
  }

  return value;
}

function readOptionalStringArray(
  raw: Record<string, unknown>,
  key: string,
  label: string,
): string[] | undefined {
  if (!(key in raw)) {
    return undefined;
  }

  const value = raw[key];
  if (!Array.isArray(value)) {
    throw new ProjectToolkitError(`${label} must be an array of strings`);
  }

  const result = value.map((entry, index) => {
    if (typeof entry !== "string") {
      throw new ProjectToolkitError(`${label}[${index}] must be a string`);
    }

    const normalized = entry.trim();
    if (!normalized) {
      throw new ProjectToolkitError(`${label}[${index}] must be a non-empty string`);
    }

    return normalized;
  });

  return result.length > 0 ? result : undefined;
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
