#!/usr/bin/env node

import path from "node:path";
import process from "node:process";
import { CodexSdkAdapter } from "../adapters/codex-sdk/adapter.js";
import { getAuthStatus } from "../core/auth.js";
import { loadRepoKitConfig } from "../core/config.js";
import { runDevWrapper } from "../core/dev-wrapper.js";
import { RepoKitError } from "../core/errors.js";
import { getPackageRoot } from "../core/package-root.js";
import { collectRepoContext } from "../core/repo-context.js";
import { createSessionLog } from "../core/session-log.js";
import { discoverSkills, loadSkill } from "../core/skills.js";
import type {
    LoadedSkill,
    PlanExecutionResult,
    RepoContext,
    SkillSummary,
    TaskExecutionResult,
} from "../core/types.js";

async function main(): Promise<void> {
  const args = process.argv.slice(2);

  if (args.length === 0 || args.includes("--help") || args.includes("-h")) {
    printUsage();
    process.exit(args.length === 0 ? 1 : 0);
  }

  const packageRoot = getPackageRoot();
  const skillsRoot = path.join(packageRoot, "skills");

  switch (args[0]) {
    case "skills":
      await handleSkillsCommand(args.slice(1), skillsRoot);
      return;
    case "plan":
      await handleAgentCommand("plan", args.slice(1), skillsRoot);
      return;
    case "run":
      await handleAgentCommand("run", args.slice(1), skillsRoot);
      return;
    case "dev":
      await handleDevCommand(args.slice(1));
      return;
    case "auth":
      await handleAuthCommand(args.slice(1));
      return;
    default:
      throw new RepoKitError(`Unknown command: ${args[0]}`);
  }
}

async function handleSkillsCommand(args: string[], skillsRoot: string): Promise<void> {
  if (args[0] !== "list" || args.length !== 1) {
    throw new RepoKitError("Usage: repo-kit skills list");
  }

  const skills = await discoverSkills(skillsRoot);
  printSkillSummaries(skills);
}

async function handleAgentCommand(mode: "plan" | "run", args: string[], skillsRoot: string): Promise<void> {
  const skillId = args[0];
  if (!skillId || args.length !== 1) {
    throw new RepoKitError(`Usage: repo-kit ${mode} <skill-id>`);
  }

  const cwd = process.cwd();
  const config = await loadRepoKitConfig(cwd);
  const skill = await loadSkill(skillsRoot, skillId);
  const repoContext = await collectRepoContext(cwd);

  if (!repoContext.gitRoot) {
    throw new RepoKitError("Current working directory must be inside a Git repository");
  }

  const sessionLog = await createSessionLog({
    cwd: repoContext.cwd,
    gitRoot: repoContext.gitRoot,
    config,
  });

  await appendSessionContextEvent({
    sessionLog,
    sessionKind: mode,
    repoContext,
    config,
    skillId: skill.id,
  });

  await sessionLog.append({
    source: "cli",
    eventType: `${mode}.started`,
    level: "info",
    skillId: skill.id,
    message: `Starting ${mode}`,
    payload: {
      projectName: config.project?.name ?? null,
    },
  });

  const adapter = new CodexSdkAdapter();

  try {
    if (mode === "plan") {
      const result = await adapter.plan({ skill, repoContext });
      await sessionLog.append({
        source: "cli",
        eventType: "plan.completed",
        level: "info",
        skillId: skill.id,
        message: "Plan completed",
        payload: {
          threadId: result.threadId,
          changedFiles: result.changedFiles,
          commands: result.commands,
        },
      });
      printPlanResult(skill, repoContext, result, sessionLog.filePath);
      return;
    }

    const result = await adapter.run({ skill, repoContext });
    await sessionLog.append({
      source: "cli",
      eventType: "run.completed",
      level: "info",
      skillId: skill.id,
      message: "Run completed",
      payload: {
        threadId: result.threadId,
        changedFiles: result.changedFiles,
        commands: result.commands,
      },
    });
    printRunResult(skill, repoContext, result, sessionLog.filePath);
  } catch (error) {
    await sessionLog.append({
      source: "cli",
      eventType: `${mode}.failed`,
      level: "error",
      skillId: skill.id,
      message: getErrorMessage(error),
    });
    throw error;
  }
}

async function handleDevCommand(args: string[]): Promise<void> {
  const cwd = process.cwd();
  const config = await loadRepoKitConfig(cwd);
  const repoContext = await collectRepoContext(cwd);
  const sessionLog = await createSessionLog({
    cwd: repoContext.cwd,
    gitRoot: repoContext.gitRoot,
    config,
  });

  await appendSessionContextEvent({
    sessionLog,
    sessionKind: "dev",
    repoContext,
    config,
    commandArgs: args,
  });

  console.log(`Session log: ${sessionLog.filePath}`);

  const exitCode = await runDevWrapper({
    args,
    config,
    repoContext,
    sessionLog,
  });

  process.exitCode = exitCode;
}

async function handleAuthCommand(args: string[]): Promise<void> {
  if (args[0] !== "status" || args.length !== 1) {
    throw new RepoKitError("Usage: repo-kit auth status");
  }

  const auth = getAuthStatus();
  console.log(`Auth source: ${auth.source}`);
  console.log(`Status: ${auth.available ? "available" : "missing"}`);
  console.log(
    auth.available
      ? "repo-kit can authenticate to Codex with the configured API key."
      : "Set OPENAI_API_KEY to enable plan/run commands.",
  );
}

function printSkillSummaries(skills: SkillSummary[]): void {
  if (skills.length === 0) {
    console.log("No skills found.");
    return;
  }

  const rows = skills.map((skill) => ({
    id: skill.id,
    title: skill.title ?? "-",
    valid: skill.valid ? "yes" : "no",
    notes: skill.errors.join("; ") || "-",
  }));

  const widths = {
    id: Math.max("ID".length, ...rows.map((row) => row.id.length)),
    title: Math.max("TITLE".length, ...rows.map((row) => row.title.length)),
    valid: Math.max("VALID".length, ...rows.map((row) => row.valid.length)),
  };

  console.log(
    `${pad("ID", widths.id)}  ${pad("TITLE", widths.title)}  ${pad("VALID", widths.valid)}  NOTES`,
  );
  for (const row of rows) {
    console.log(
      `${pad(row.id, widths.id)}  ${pad(row.title, widths.title)}  ${pad(row.valid, widths.valid)}  ${row.notes}`,
    );
  }
}

function printPlanResult(
  skill: LoadedSkill,
  repoContext: RepoContext,
  result: PlanExecutionResult,
  sessionLogPath: string,
): void {
  console.log(`Skill: ${skill.id}`);
  console.log(`Working directory: ${repoContext.cwd}`);
  if (result.plan) {
    console.log("");
    console.log(`Summary: ${result.plan.summary}`);
    console.log("");
    console.log("Steps:");
    for (const step of result.plan.steps) {
      console.log(`- ${step}`);
    }
    console.log("");
    console.log("Risks:");
    if (result.plan.risks.length === 0) {
      console.log("- none");
    } else {
      for (const risk of result.plan.risks) {
        console.log(`- ${risk}`);
      }
    }
  } else {
    console.log("");
    console.log(result.finalResponse);
  }

  printExecutionFooter(result, sessionLogPath);
}

function printRunResult(
  skill: LoadedSkill,
  repoContext: RepoContext,
  result: TaskExecutionResult,
  sessionLogPath: string,
): void {
  console.log(`Skill: ${skill.id}`);
  console.log(`Working directory: ${repoContext.cwd}`);
  console.log("");
  console.log(result.finalResponse);
  printExecutionFooter(result, sessionLogPath);
}

function printExecutionFooter(result: PlanExecutionResult | TaskExecutionResult, sessionLogPath: string): void {
  console.log("");
  console.log(`Thread ID: ${result.threadId ?? "n/a"}`);
  console.log(`Session log: ${sessionLogPath}`);

  if (result.changedFiles.length > 0) {
    console.log("Changed files:");
    for (const changedFile of result.changedFiles) {
      console.log(`- ${changedFile}`);
    }
  }

  if (result.commands.length > 0) {
    console.log("Commands:");
    for (const command of result.commands) {
      console.log(`- ${command}`);
    }
  }

  if (result.usage) {
    console.log(
      `Usage: input=${result.usage.inputTokens}, cached=${result.usage.cachedInputTokens}, output=${result.usage.outputTokens}`,
    );
  }
}

function printUsage(): void {
  console.log(`repo-kit skills list
repo-kit plan <skill-id>
repo-kit run <skill-id>
repo-kit dev [--] <command...>
repo-kit auth status`);
}

function pad(value: string, width: number): string {
  return value.padEnd(width, " ");
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

async function appendSessionContextEvent(options: {
  sessionLog: Awaited<ReturnType<typeof createSessionLog>>;
  sessionKind: "plan" | "run" | "dev";
  repoContext: RepoContext;
  config: Awaited<ReturnType<typeof loadRepoKitConfig>>;
  skillId?: string;
  commandArgs?: string[];
}): Promise<void> {
  const { sessionLog, sessionKind, repoContext, config, skillId, commandArgs } = options;

  const event = {
    source: "cli",
    eventType: "session.context",
    level: "info" as const,
    message: `Captured ${sessionKind} session context`,
    payload: {
      sessionKind,
      projectName: config.project?.name ?? null,
      gitBranch: repoContext.gitBranch,
      gitRoot: repoContext.gitRoot,
      topLevelEntries: repoContext.topLevelEntries.slice(0, 20),
      commandArgs: commandArgs?.slice(0, 20) ?? [],
      filePreviews: repoContext.filePreviews.slice(0, 3).map((preview) => ({
        path: preview.path,
        preview: truncate(preview.preview, 400),
      })),
    },
  };

  if (skillId) {
    await sessionLog.append({
      ...event,
      skillId,
    });
    return;
  }

  await sessionLog.append(event);
}

function truncate(value: string, maxLength: number): string {
  return value.length > maxLength ? `${value.slice(0, maxLength)}...` : value;
}

try {
  await main();
} catch (error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  console.error(`error: ${message}`);
  process.exit(error instanceof RepoKitError ? error.exitCode : 1);
}
