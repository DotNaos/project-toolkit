import {
    Codex,
    type FileChangeItem,
    type RunResult,
    type ThreadItem,
    type ThreadOptions,
} from "@openai/codex-sdk";
import { getApiKey } from "../../core/auth.js";
import { ProjectToolkitError } from "../../core/errors.js";
import { renderRepoContext } from "../../core/repo-context.js";
import type {
    AgentAdapter,
    ExecutionUsage,
    PlanExecutionResult,
    PlannedChanges,
    SkillExecutionInput,
    TaskExecutionResult,
} from "../../core/types.js";

const PLAN_SCHEMA = {
  type: "object",
  properties: {
    summary: { type: "string" },
    steps: {
      type: "array",
      items: { type: "string" },
    },
    risks: {
      type: "array",
      items: { type: "string" },
    },
  },
  required: ["summary", "steps", "risks"],
  additionalProperties: false,
} as const;

export class CodexSdkAdapter implements AgentAdapter {
  private readonly codex: Codex;

  constructor(apiKey = getApiKey()) {
    if (!apiKey) {
      throw new ProjectToolkitError("OPENAI_API_KEY is required");
    }

    this.codex = new Codex({ apiKey });
  }

  async plan(input: SkillExecutionInput): Promise<PlanExecutionResult> {
    const thread = this.codex.startThread(buildThreadOptions(input, "plan"));
    const turn = await thread.run(buildPlanPrompt(input), { outputSchema: PLAN_SCHEMA });

    return {
      mode: "plan",
      plan: parsePlan(turn.finalResponse),
      ...summarizeTurn(thread.id, turn),
    };
  }

  async run(input: SkillExecutionInput): Promise<TaskExecutionResult> {
    const thread = this.codex.startThread(buildThreadOptions(input, "run"));
    const turn = await thread.run(buildRunPrompt(input));

    return {
      mode: "run",
      ...summarizeTurn(thread.id, turn),
    };
  }
}

function buildThreadOptions(input: SkillExecutionInput, mode: "plan" | "run"): ThreadOptions {
  return {
    workingDirectory: input.repoContext.cwd,
    additionalDirectories: [input.skill.rootDir],
    sandboxMode: mode === "plan" ? "read-only" : "workspace-write",
    approvalPolicy: "never",
    networkAccessEnabled: mode === "run",
  };
}

function buildPlanPrompt(input: SkillExecutionInput): string {
  return [
    "You are running inside project-toolkit planning mode.",
    "Produce a concise, non-destructive implementation plan for the selected skill.",
    "Do not modify files, generate patches, or execute destructive commands.",
    "Stay within the selected skill's scope and the repository state shown below.",
    "",
    renderSkillSection(input),
    renderRepoContext(input.repoContext),
  ].join("\n");
}

function buildRunPrompt(input: SkillExecutionInput): string {
  return [
    "You are running inside project-toolkit execution mode.",
    "Apply the selected skill to the repository in the current working directory.",
    "Keep the change set minimal, production-ready, and within the selected skill's scope.",
    "If you are blocked, explain the blocker clearly in the final response.",
    "",
    renderSkillSection(input),
    renderRepoContext(input.repoContext),
  ].join("\n");
}

function renderSkillSection(input: SkillExecutionInput): string {
  return [
    "Selected skill:",
    `- id: ${input.skill.id}`,
    `- title: ${input.skill.title}`,
    `- description: ${input.skill.description ?? "n/a"}`,
    `- skill root: ${input.skill.rootDir}`,
    `- prompt file: ${input.skill.promptPath ?? "embedded"}`,
    "",
    "Skill instructions:",
    input.skill.prompt,
  ].join("\n");
}

function parsePlan(source: string): PlannedChanges | null {
  try {
    const parsed = JSON.parse(source) as Partial<PlannedChanges>;
    if (
      typeof parsed.summary === "string" &&
      Array.isArray(parsed.steps) &&
      parsed.steps.every((value) => typeof value === "string") &&
      Array.isArray(parsed.risks) &&
      parsed.risks.every((value) => typeof value === "string")
    ) {
      return {
        summary: parsed.summary,
        steps: parsed.steps,
        risks: parsed.risks,
      };
    }
  } catch {
    return null;
  }

  return null;
}

function summarizeTurn(threadId: string | null, turn: RunResult): {
  threadId: string | null;
  finalResponse: string;
  changedFiles: string[];
  commands: string[];
  usage: ExecutionUsage | null;
} {
  return {
    threadId,
    finalResponse: turn.finalResponse,
    changedFiles: collectChangedFiles(turn.items),
    commands: turn.items
      .filter((item): item is Extract<ThreadItem, { type: "command_execution" }> => item.type === "command_execution")
      .map((item) => item.command),
    usage: turn.usage
      ? {
          inputTokens: turn.usage.input_tokens,
          cachedInputTokens: turn.usage.cached_input_tokens,
          outputTokens: turn.usage.output_tokens,
        }
      : null,
  };
}

function collectChangedFiles(items: ThreadItem[]): string[] {
  return items
    .filter((item): item is FileChangeItem => item.type === "file_change")
    .flatMap((item) => item.changes.map((change) => change.path))
    .filter((value, index, values) => values.indexOf(value) === index);
}
