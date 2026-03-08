import { spawn } from "node:child_process";
import process from "node:process";
import { ProjectToolkitError } from "./errors.js";
import type { ProjectToolkitConfig, RepoContext, SessionLog, SessionLogLevel } from "./types.js";

interface DevWrapperOptions {
  args: string[];
  config: ProjectToolkitConfig;
  repoContext: RepoContext;
  sessionLog: SessionLog;
}

interface ResolvedDevCommand {
  command: string;
  args: string[];
  displayCommand: string;
  shell: boolean;
}

export async function runDevWrapper(options: DevWrapperOptions): Promise<number> {
  const resolved = resolveDevCommand(options.args, options.config);
  const stdoutBuffer = { value: "" };
  const stderrBuffer = { value: "" };

  await options.sessionLog.append({
    source: "dev-wrapper",
    eventType: "command.started",
    level: "info",
    command: resolved.displayCommand,
    message: "Starting dev command",
    payload: {
      shell: resolved.shell,
      args: resolved.args,
    },
  });

  return await new Promise<number>((resolve, reject) => {
    const child = spawn(resolved.command, resolved.args, {
      cwd: options.repoContext.cwd,
      env: process.env,
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

function resolveDevCommand(args: string[], config: ProjectToolkitConfig): ResolvedDevCommand {
  const normalizedArgs = args[0] === "--" ? args.slice(1) : args;
  const explicitCommand = normalizedArgs[0];

  if (explicitCommand) {
    return {
      command: explicitCommand,
      args: normalizedArgs.slice(1),
      displayCommand: normalizedArgs.join(" "),
      shell: false,
    };
  }

  const configuredCommand = config.dev?.command;
  if (configuredCommand) {
    return {
      command: configuredCommand,
      args: [],
      displayCommand: configuredCommand,
      shell: true,
    };
  }

  throw new ProjectToolkitError("Usage: pkit dev [--] <command...> or configure .project-toolkit/config.yaml dev.command");
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
