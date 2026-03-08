# Architecture

`repo-kit` v1 is a small TypeScript CLI with one agent backend and filesystem-based skill discovery.

## Source Layout

- `src/cli/`
    - command parsing
    - terminal output
- `src/core/`
    - shared types
    - skill discovery and loading
    - minimal repository context collection
    - authentication checks
    - repository-local config loading
    - session JSONL logging
    - dev command wrapper for local workflows
- `src/adapters/codex-sdk/`
    - the only agent backend in v1
    - maps `plan` and `run` into Codex SDK thread execution
- `skills/`
    - bundled skills and supporting assets
- `docs/`
    - usage and publishing notes

## Execution Flow

1. The CLI resolves the packaged `skills/` directory.
2. `skills list` inspects each first-level skill directory and reports whether it is runnable.
3. `plan` and `run` load a single skill, collect minimal context from the current working directory, and require a Git repository.
4. `plan`, `run`, and `dev` load `.repo-kit/config.yaml` when present.
5. `plan` and `run` create a per-session JSONL log file before invoking the adapter.
6. The Codex adapter starts a thread with the current repository as the working directory and the selected skill directory exposed via `additionalDirectories`.
7. `plan` uses read-only sandboxing and a structured JSON plan schema.
8. `run` uses workspace-write sandboxing and returns the agent response plus basic execution summaries.
9. `dev` wraps either an explicit subprocess or a configured shell command, tees terminal output, and appends structured output events to the same JSONL session log.

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

- Configuration is optional and repository-local at `.repo-kit/config.yaml`.
- Session logs are JSONL files written to `logs/repo-kit/` by default, or to `logs.dir` when configured.
- Each session log entry includes a timestamp, random session id, cwd, git root, event source, event type, severity, and optional command or skill metadata.
- The CLI keeps human-readable terminal output separate from structured logs so local workflows stay friendly while automation stays inspectable.
