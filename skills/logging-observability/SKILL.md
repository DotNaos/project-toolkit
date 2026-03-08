# Logging and observability baseline

Establish a lightweight observability baseline for CLI and Node.js service workflows.

## Goals

- Prefer append-only JSONL session logs for local tooling and automation traces.
- Keep terminal output human-readable while recording structured events in parallel.
- Capture enough context to debug failures quickly: timestamp, session id, cwd, git root, source, event type, severity, and command metadata.
- Make log destinations configurable, but keep a safe default under `logs/`.

## When applying this skill

- Add structured logging with minimal dependencies.
- Keep event schemas small and versionable.
- Use line-oriented events for command stdout and stderr when full tracing is unnecessary.
- Document how developers can override paths and commands in repository-local config.

## References

- `references/jsonl-event-schema.md`
- `references/node-service-pattern.md`
- `references/observability-roadmap.md`
