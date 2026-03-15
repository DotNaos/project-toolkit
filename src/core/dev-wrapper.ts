import { spawn } from "node:child_process";
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { ProjectToolkitError } from "./errors.js";
import type { ProjectToolkitConfig, RepoContext, SessionLog, SessionLogLevel } from "./types.js";

interface DevWrapperOptions {
  args: string[];
  config: ProjectToolkitConfig;
  repoContext: RepoContext;
  sessionLog: SessionLog;
}

type ResolvedDevCommandSource = "explicit" | "config-args" | "config-command";

interface ResolvedDevCommand {
  command: string;
  args: string[];
  displayCommand: string;
  shell: boolean;
  source: ResolvedDevCommandSource;
  env?: Record<string, string>;
  notes?: string[];
}

export async function runDevWrapper(options: DevWrapperOptions): Promise<number> {
  const resolved = await resolveDevCommand(options.args, options.config, options.repoContext);
  const stdoutBuffer = { value: "" };
  const stderrBuffer = { value: "" };

  for (const note of resolved.notes ?? []) {
    process.stdout.write(`${note}\n`);
  }

  const startPayload: Record<string, boolean | string[] | Record<string, string>> = {
    shell: resolved.shell,
    args: resolved.args,
  };
  if (resolved.env) {
    startPayload.env = resolved.env;
  }

  await options.sessionLog.append({
    source: "dev-wrapper",
    eventType: "command.started",
    level: "info",
    command: resolved.displayCommand,
    message: "Starting dev command",
    payload: startPayload,
  });

  return await new Promise<number>((resolve, reject) => {
    const child = spawn(resolved.command, resolved.args, {
      cwd: options.repoContext.cwd,
      env: { ...process.env, ...(resolved.env ?? {}) },
      shell: resolved.shell,
      stdio: ["inherit", "pipe", "pipe"],
    });

    child.stdout?.on("data", (chunk: Buffer | string) => {
      forwardOutput({
        chunk,
        buffer: stdoutBuffer,
        writer: process.stdout,
        sessionLog: options.sessionLog,
        command: resolved.displayCommand,
        source: "stdout",
        level: "info",
      });
    });

    child.stderr?.on("data", (chunk: Buffer | string) => {
      forwardOutput({
        chunk,
        buffer: stderrBuffer,
        writer: process.stderr,
        sessionLog: options.sessionLog,
        command: resolved.displayCommand,
        source: "stderr",
        level: "error",
      });
    });

    child.once("error", async (error) => {
      flushBuffer(stdoutBuffer, options.sessionLog, resolved.displayCommand, "stdout", "info");
      flushBuffer(stderrBuffer, options.sessionLog, resolved.displayCommand, "stderr", "error");
      await options.sessionLog.append({
        source: "dev-wrapper",
        eventType: "command.failed",
        level: "error",
        command: resolved.displayCommand,
        message: error.message,
      });
      reject(new ProjectToolkitError(`Failed to start dev command: ${resolved.displayCommand}`));
    });

    child.once("close", async (code, signal) => {
      flushBuffer(stdoutBuffer, options.sessionLog, resolved.displayCommand, "stdout", "info");
      flushBuffer(stderrBuffer, options.sessionLog, resolved.displayCommand, "stderr", "error");

      const exitCode = code ?? (signal ? 1 : 0);
      await options.sessionLog.append({
        source: "dev-wrapper",
        eventType: "command.completed",
        level: exitCode === 0 ? "info" : "error",
        command: resolved.displayCommand,
        message: `Dev command exited with code ${exitCode}`,
        payload: {
          exitCode,
          signal,
        },
      });

      resolve(exitCode);
    });
  });
}

async function resolveDevCommand(
  args: string[],
  config: ProjectToolkitConfig,
  repoContext: RepoContext,
): Promise<ResolvedDevCommand> {
  const baseCommand = resolveBaseDevCommand(args, config);
  const router = config.dev?.router;
  if (!router) {
    return baseCommand;
  }

  const baseName = deriveRouterBaseName(config, repoContext);
  if (router.mode === "portless") {
    return buildPortlessCommand(baseCommand, baseName, repoContext);
  }

  return await buildDockportlessCommand(baseCommand, baseName, repoContext);
}

function resolveBaseDevCommand(args: string[], config: ProjectToolkitConfig): ResolvedDevCommand {
  const normalizedArgs = args[0] === "--" ? args.slice(1) : args;
  const explicitCommand = normalizedArgs[0];
  if (explicitCommand) {
    return {
      command: explicitCommand,
      args: normalizedArgs.slice(1),
      displayCommand: normalizedArgs.join(" "),
      shell: false,
      source: "explicit",
    };
  }

  const configuredArgs = config.dev?.args;
  if (configuredArgs && configuredArgs.length > 0) {
    return {
      command: configuredArgs[0]!,
      args: configuredArgs.slice(1),
      displayCommand: configuredArgs.join(" "),
      shell: false,
      source: "config-args",
    };
  }

  const configuredCommand = config.dev?.command;
  if (configuredCommand) {
    return {
      command: configuredCommand,
      args: [],
      displayCommand: configuredCommand,
      shell: true,
      source: "config-command",
    };
  }

  throw new ProjectToolkitError(
    "Usage: pkit dev [--] <command...> or configure .project-toolkit/config.yaml dev.command/dev.args",
  );
}

function buildPortlessCommand(
  command: ResolvedDevCommand,
  baseName: string,
  repoContext: RepoContext,
): ResolvedDevCommand {
  const notes = [
    `Portless base URL: http://${baseName}.localhost:1355`,
  ];

  if (repoContext.gitBranch) {
    notes.push(`Linked worktrees keep the same base name and add the branch as a subdomain automatically.`);
  }

  if (command.source === "config-command") {
    const wrappedCommand = `portless run --name ${quoteForShell(baseName)} ${command.displayCommand}`;
    return {
      command: wrappedCommand,
      args: [],
      displayCommand: wrappedCommand,
      shell: true,
      source: command.source,
      notes,
    };
  }

  return {
    command: "portless",
    args: ["run", "--name", baseName, command.command, ...command.args],
    displayCommand: `portless run --name ${baseName} ${command.displayCommand}`,
    shell: false,
    source: command.source,
    notes,
  };
}

async function buildDockportlessCommand(
  command: ResolvedDevCommand,
  baseName: string,
  repoContext: RepoContext,
): Promise<ResolvedDevCommand> {
  if (hasComposeProjectOverride(command)) {
    throw new ProjectToolkitError(
      "Do not hardcode Docker Compose project names when dev.router.mode is 'dockportless'; project-toolkit manages COMPOSE_PROJECT_NAME automatically.",
    );
  }

  if (!looksLikeComposeCommand(command)) {
    throw new ProjectToolkitError(
      "dev.router.mode 'dockportless' only supports compose-compatible commands such as 'docker compose ...' or 'docker-compose ...'.",
    );
  }

  const projectName = await deriveDockportlessProjectName(baseName, repoContext);
  const notes = [
    `Dockportless project: ${projectName}`,
    `Compose project: ${projectName}`,
    `Service URLs: http://<service>.${projectName}.localhost:7355`,
  ];

  if (command.source === "config-command") {
    const wrappedCommand = `COMPOSE_PROJECT_NAME=${quoteForShell(projectName)} dockportless run ${quoteForShell(projectName)} ${command.displayCommand}`;
    return {
      command: wrappedCommand,
      args: [],
      displayCommand: wrappedCommand,
      shell: true,
      source: command.source,
      env: {
        COMPOSE_PROJECT_NAME: projectName,
      },
      notes,
    };
  }

  return {
    command: "dockportless",
    args: ["run", projectName, command.command, ...command.args],
    displayCommand: `COMPOSE_PROJECT_NAME=${projectName} dockportless run ${projectName} ${command.displayCommand}`,
    shell: false,
    source: command.source,
    env: {
      COMPOSE_PROJECT_NAME: projectName,
    },
    notes,
  };
}

function deriveRouterBaseName(config: ProjectToolkitConfig, repoContext: RepoContext): string {
  const configuredName = config.dev?.router?.name;
  if (configuredName) {
    return slugify(configuredName);
  }

  if (config.project?.name) {
    return slugify(config.project.name);
  }

  if (repoContext.gitRoot) {
    return slugify(path.basename(repoContext.gitRoot));
  }

  return slugify(path.basename(repoContext.cwd));
}

async function deriveDockportlessProjectName(baseName: string, repoContext: RepoContext): Promise<string> {
  const gitRoot = repoContext.gitRoot ?? repoContext.cwd;
  if (!(await isLinkedWorktree(gitRoot))) {
    return baseName;
  }

  const branchSlug = slugify(repoContext.gitBranch ?? "");
  if (branchSlug && branchSlug !== baseName) {
    return `${baseName}-${branchSlug}`;
  }

  const worktreeSlug = slugify(path.basename(gitRoot));
  if (worktreeSlug && worktreeSlug !== baseName) {
    return `${baseName}-${worktreeSlug}`;
  }

  return `${baseName}-worktree`;
}

async function isLinkedWorktree(gitRoot: string): Promise<boolean> {
  try {
    const stats = await fs.lstat(path.join(gitRoot, ".git"));
    return stats.isFile();
  } catch {
    return false;
  }
}

function hasComposeProjectOverride(command: ResolvedDevCommand): boolean {
  if (command.source === "config-command") {
    return /(^|\s)(-p|--project-name)(\s|=)/.test(command.displayCommand);
  }

  return command.command === "docker-compose" || command.command === "podman-compose"
    ? command.args.some((arg, index) => arg === "-p" || arg === "--project-name" || arg.startsWith("--project-name="))
    : command.args.some((arg, index) => {
        if (arg === "-p" || arg === "--project-name" || arg.startsWith("--project-name=")) {
          return true;
        }

        return index > 0 && command.args[index - 1] === "compose" && (arg === "-p" || arg === "--project-name");
      });
}

function looksLikeComposeCommand(command: ResolvedDevCommand): boolean {
  if (command.source === "config-command") {
    return /\b(docker|nerdctl|podman)\s+compose\b/.test(command.displayCommand) || /\b(docker-compose|podman-compose)\b/.test(command.displayCommand);
  }

  if (command.command === "docker-compose" || command.command === "podman-compose") {
    return true;
  }

  return (command.command === "docker" || command.command === "nerdctl" || command.command === "podman") && command.args[0] === "compose";
}

function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replaceAll(/[^a-z0-9]+/g, "-").replaceAll(/^-+|-+$/g, "");
  return normalized || "project";
}

function quoteForShell(value: string): string {
  return `'${value.replaceAll("'", `'\"'\"'`)}'`;
}

function forwardOutput(options: {
  chunk: Buffer | string;
  buffer: { value: string };
  writer: NodeJS.WriteStream;
  sessionLog: SessionLog;
  command: string;
  source: "stdout" | "stderr";
  level: SessionLogLevel;
}): void {
  const text = String(options.chunk);
  options.writer.write(text);
  options.buffer.value += text;

  flushCompletedLines(options.buffer, (line) => {
    options.sessionLog.append({
      source: options.source,
      eventType: "command.output",
      level: options.level,
      command: options.command,
      message: line,
    }).catch(() => undefined);
  });
}

function flushBuffer(
  buffer: { value: string },
  sessionLog: SessionLog,
  command: string,
  source: "stdout" | "stderr",
  level: SessionLogLevel,
): void {
  const remainder = buffer.value.trimEnd();
  if (!remainder) {
    buffer.value = "";
    return;
  }

  buffer.value = "";
  sessionLog.append({
    source,
    eventType: "command.output",
    level,
    command,
    message: remainder,
  }).catch(() => undefined);
}

function flushCompletedLines(buffer: { value: string }, onLine: (line: string) => void): void {
  while (true) {
    const newlineIndex = buffer.value.indexOf("\n");
    if (newlineIndex === -1) {
      return;
    }

    const line = buffer.value.slice(0, newlineIndex).replace(/\r$/, "");
    buffer.value = buffer.value.slice(newlineIndex + 1);
    onLine(line);
  }
}
