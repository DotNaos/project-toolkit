import fs from "node:fs/promises";
import path from "node:path";
import { parse as parseYaml } from "yaml";
import { RepoKitError } from "./errors.js";
import type { RepoKitConfig } from "./types.js";

const CONFIG_RELATIVE_PATH = path.join(".repo-kit", "config.yaml");

export async function loadRepoKitConfig(cwd: string): Promise<RepoKitConfig> {
  const configPath = path.join(cwd, CONFIG_RELATIVE_PATH);

  let source: string;
  try {
    source = await fs.readFile(configPath, "utf8");
  } catch (error) {
    if (isNodeError(error) && error.code === "ENOENT") {
      return {};
    }

    throw new RepoKitError(`Failed to read ${CONFIG_RELATIVE_PATH}: ${getErrorMessage(error)}`);
  }

  let parsed: unknown;
  try {
    parsed = parseYaml(source);
  } catch (error) {
    throw new RepoKitError(`Invalid ${CONFIG_RELATIVE_PATH}: ${getErrorMessage(error)}`);
  }

  if (parsed == null) {
    return {};
  }

  if (!isRecord(parsed)) {
    throw new RepoKitError(`${CONFIG_RELATIVE_PATH} must contain a YAML object`);
  }

  return validateConfig(parsed, CONFIG_RELATIVE_PATH);
}

function validateConfig(raw: Record<string, unknown>, label: string): RepoKitConfig {
  const dev = readNestedObject(raw, "dev", label);
  const logs = readNestedObject(raw, "logs", label);
  const project = readNestedObject(raw, "project", label);

  const devCommand = readOptionalString(dev, "command", `${label}.dev.command`);
  const logsDir = readOptionalString(logs, "dir", `${label}.logs.dir`);
  const projectName = readOptionalString(project, "name", `${label}.project.name`);

  const config: RepoKitConfig = {};

  if (devCommand) {
    config.dev = { command: devCommand };
  }

  if (logsDir) {
    config.logs = { dir: logsDir };
  }

  if (projectName) {
    config.project = { name: projectName };
  }

  return config;
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
    throw new RepoKitError(`${label}.${key} must be an object`);
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
    throw new RepoKitError(`${label} must be a string`);
  }

  const normalized = value.trim();
  return normalized.length > 0 ? normalized : undefined;
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
