# Architecture

`project-toolkit` is transitioning to a hybrid CLI:

- Go + Cobra becomes the primary product-facing CLI for install/update flows, workspaces, worktrees, configuration, and shell completions.
- TypeScript is narrowed toward the Codex-backed agent adapter and the existing implementation that the Go frontend bridges to during migration.

## Source Layout

- `cmd/pkit/` — Go entrypoint for the new hybrid CLI
- `internal/cli/` — Cobra command tree and subcommand wiring
- `internal/nodebridge/` — migration bridge that forwards command execution into the existing TypeScript CLI
- `src/cli/` — existing TypeScript CLI implementation
- `src/core/` — current TypeScript business logic for project/workspace/worktree flows
- `src/adapters/codex-sdk/` — isolated TypeScript Codex adapter target
- `skills/` — bundled skills and supporting assets
- `docs/` — usage and publishing notes

## Execution Flow

1. The Go/Cobra frontend defines the long-term command surface and owns native shell completion generation.
2. During migration, most commands forward to the existing TypeScript CLI through `internal/nodebridge/`.
3. The TypeScript CLI resolves the packaged `skills/` directory.
4. `skills list` inspects each first-level skill directory and reports whether it is runnable.
5. `project init` scaffolds `.project-toolkit/config.yaml` and `.project-toolkit/base.code-workspace` as the stable in-repo foundation for later worktree and workspace commands.
6. `project workspace generate` reads the base workspace as JSONC, replaces the `folders` array, writes a generated workspace file outside the repo by default, and applies matching shared-file symlinks for the chosen workspace/root.
7. `project worktree create` allocates a managed worktree path under toolkit state, runs `git worktree add`, and then reuses workspace generation to attach a generated workspace file and shared links.
8. `plan` and `run` remain the TypeScript-owned bridge into the Codex adapter.
9. `plan`, `run`, and `dev` load `.project-toolkit/config.yaml` when present.
10. `plan` and `run` create a per-session JSONL log file before invoking the adapter.
11. The Codex adapter starts a thread with the current repository as the working directory and the selected skill directory exposed via `additionalDirectories`.
12. `plan` uses read-only sandboxing and a structured JSON plan schema.
13. `run` uses workspace-write sandboxing and returns the agent response plus basic execution summaries.
14. `dev` wraps either an explicit subprocess or a configured `dev.args` / `dev.command`, can route it through Portless or Dockportless, tees terminal output, and appends structured output events to the same JSONL session log.

## Skill Compatibility

The loader prefers normalized skills:

- `skill.yaml`
- `prompt.md`
- optional `assets/`, `templates/`, `scripts/`

To avoid rewriting the current catalog, v1 also accepts:

- `SKILL.md`
- a single markdown file fallback for older prompt-only entries

Directories without a prompt definition are intentionally left invalid instead of guessing at runnable behavior from raw assets.

## Boundary Direction

- Go owns the user-facing command tree, install/update ergonomics, shell completions, and long-term general CLI workflows.
- TypeScript owns the Codex-specific adapter path and the pre-existing implementation during migration.
- The bridge layer is transitional and should shrink as general commands are reimplemented in Go.

## Logging and config MVP

- Configuration is optional and repository-local at `.project-toolkit/config.yaml`.
- `project init` writes both the config scaffold and `.project-toolkit/base.code-workspace` so future generated workspaces can replace only the active `folders` section while keeping a stable base file in the repo.
- `project workspace generate` defaults its output to `~/.project-toolkit/projects/<project-key>/workspaces/<name>.code-workspace`, which keeps generated state outside the repository while preserving an editable base file in-repo.
- `project worktree create` defaults worktrees to `~/.project-toolkit/projects/<project-key>/worktrees/<name>`, so parallel agent branches live outside the main repository while still sharing selected ignored files from the source repo.
- `dev.router.mode: portless` uses Portless-managed named routing for local processes, so worktrees keep stable `.localhost` URLs without reserving fixed ports.
- `dev.router.mode: dockportless` uses Dockportless for compose-compatible Docker workflows and exports a worktree-specific `COMPOSE_PROJECT_NAME` so containers, networks, and volumes stay isolated.
- Session logs are JSONL files written to `logs/project-toolkit/` by default, or to `logs.dir` when configured.
- Each session log entry includes a timestamp, random session id, cwd, git root, event source, event type, severity, and optional command or skill metadata.
- The CLI keeps human-readable terminal output separate from structured logs so local workflows stay friendly while automation stays inspectable.
