# Architecture

`project-toolkit` v1 starts as a TypeScript CLI with one agent backend and filesystem-based skill discovery, with room to expand into broader project operations.

## Source Layout

- `src/cli/` — command parsing and terminal output
- `src/core/` — shared types, config loading, project scaffolding, workspace generation, managed worktree creation, session logging, and local workflow helpers
- `src/adapters/codex-sdk/` — the only agent backend in v1, mapping `plan` and `run` into Codex SDK thread execution
- `skills/` — bundled skills and supporting assets
- `docs/` — usage and publishing notes

## Execution Flow

1. The CLI resolves the packaged `skills/` directory.
2. `skills list` inspects each first-level skill directory and reports whether it is runnable.
3. `project init` scaffolds `.project-toolkit/config.yaml` and `.project-toolkit/base.code-workspace` as the stable in-repo foundation for later worktree and workspace commands.
4. `project workspace generate` reads the base workspace as JSONC, replaces the `folders` array, writes a generated workspace file outside the repo by default, and applies matching shared-file symlinks for the chosen workspace/root.
5. `project worktree create` allocates a managed worktree path under toolkit state, runs `git worktree add`, and then reuses workspace generation to attach a generated workspace file and shared links.
6. `plan` and `run` load a single skill, collect minimal context from the current working directory, and require a Git repository.
7. `plan`, `run`, and `dev` load `.project-toolkit/config.yaml` when present.
8. `plan` and `run` create a per-session JSONL log file before invoking the adapter.
9. The Codex adapter starts a thread with the current repository as the working directory and the selected skill directory exposed via `additionalDirectories`.
10. `plan` uses read-only sandboxing and a structured JSON plan schema.
11. `run` uses workspace-write sandboxing and returns the agent response plus basic execution summaries.
12. `dev` wraps either an explicit subprocess or a configured shell command, tees terminal output, and appends structured output events to the same JSONL session log.

## Skill Compatibility

The loader prefers normalized skills:

- `skill.yaml`
- `prompt.md`
- optional `assets/`, `templates/`, `scripts/`

To avoid rewriting the current catalog, v1 also accepts:

- `SKILL.md`
- a single markdown file fallback for older prompt-only entries

Directories without a prompt definition are intentionally left invalid instead of guessing at runnable behavior from raw assets.

## Logging and config MVP

- Configuration is optional and repository-local at `.project-toolkit/config.yaml`.
- `project init` writes both the config scaffold and `.project-toolkit/base.code-workspace` so future generated workspaces can replace only the active `folders` section while keeping a stable base file in the repo.
- `project workspace generate` defaults its output to `~/.project-toolkit/projects/<project-key>/workspaces/<name>.code-workspace`, which keeps generated state outside the repository while preserving an editable base file in-repo.
- `project worktree create` defaults worktrees to `~/.project-toolkit/projects/<project-key>/worktrees/<name>`, so parallel agent branches live outside the main repository while still sharing selected ignored files from the source repo.
- Session logs are JSONL files written to `logs/project-toolkit/` by default, or to `logs.dir` when configured.
- Each session log entry includes a timestamp, random session id, cwd, git root, event source, event type, severity, and optional command or skill metadata.
- The CLI keeps human-readable terminal output separate from structured logs so local workflows stay friendly while automation stays inspectable.
