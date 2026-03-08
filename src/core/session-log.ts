import { randomUUID } from "node:crypto";
import fs from "node:fs/promises";
import path from "node:path";
import type {
    ProjectToolkitConfig,
    SessionLog,
    SessionLogEvent,
    SessionLogEventInput,
} from "./types.js";

interface CreateSessionLogOptions {
  cwd: string;
  gitRoot: string | null;
  config: ProjectToolkitConfig;
}

export async function createSessionLog(options: CreateSessionLogOptions): Promise<SessionLog> {
  const sessionId = randomUUID();
  const logsDir = resolveLogsDir(options.cwd, options.config);
  const filePath = path.join(logsDir, `${formatTimestampForFileName(new Date())}-${sessionId}.jsonl`);

  await fs.mkdir(logsDir, { recursive: true });
  await fs.writeFile(filePath, "", "utf8");

  let writeQueue = Promise.resolve();

  return {
    sessionId,
    filePath,
    append(event: SessionLogEventInput): Promise<void> {
      writeQueue = writeQueue.then(async () => {
        const record = buildEventRecord({
          event,
          sessionId,
          cwd: options.cwd,
          gitRoot: options.gitRoot,
        });

        await fs.appendFile(filePath, `${JSON.stringify(record)}\n`, "utf8");
      });

      return writeQueue;
    },
  };
}

function resolveLogsDir(cwd: string, config: ProjectToolkitConfig): string {
  const configuredDir = config.logs?.dir;
  if (!configuredDir) {
    return path.join(cwd, "logs", "project-toolkit");
  }

  return path.isAbsolute(configuredDir) ? configuredDir : path.resolve(cwd, configuredDir);
}

function buildEventRecord(options: {
  event: SessionLogEventInput;
  sessionId: string;
  cwd: string;
  gitRoot: string | null;
}): SessionLogEvent {
  const { event, sessionId, cwd, gitRoot } = options;

  const record: SessionLogEvent = {
    timestamp: new Date().toISOString(),
    sessionId,
    source: event.source,
    eventType: event.eventType,
    level: event.level,
    cwd,
    gitRoot,
  };

  if (event.skillId) {
    record.skillId = event.skillId;
  }

  if (event.command) {
    record.command = event.command;
  }

  if (event.message) {
    record.message = event.message;
  }

  if (event.payload !== undefined) {
    record.payload = event.payload;
  }

  return record;
}

function formatTimestampForFileName(date: Date): string {
  return date.toISOString().replaceAll(/[:.]/g, "-");
}
