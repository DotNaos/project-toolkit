# repo-kit

`repo-kit` is a minimal Node.js / TypeScript CLI for running repository skills through the official Codex SDK.

v1 intentionally supports one backend only:

- Codex via `@openai/codex-sdk`
- API key authentication via `OPENAI_API_KEY`

The bundled skill catalog stays at the repository root in `skills/`.

## Requirements

- Node.js 18+
- `OPENAI_API_KEY` for `plan` and `run`
- A Git repository as the current working directory when running skills

## Install

This package is configured for GitHub Packages only.

```bash
npm config set @dotnaos:registry https://npm.pkg.github.com
export NODE_AUTH_TOKEN=YOUR_GITHUB_TOKEN
npm install @dotnaos/repo-kit
```

Use a GitHub token that can read packages for installation. The checked-in `.npmrc` already points the `@dotnaos` scope at GitHub Packages and reads the token from `NODE_AUTH_TOKEN`.

## Local Development

```bash
npm install
npm run build
node dist/cli/index.js skills list
```

## CLI Usage

```bash
repo-kit skills list
repo-kit plan <skill-id>
repo-kit run <skill-id>
repo-kit dev [--] <command...>
repo-kit auth status
```

Command behavior in v1:

- `skills list` discovers first-level directories under `skills/` and reports whether each one looks runnable.
- `plan <skill-id>` loads the selected skill, collects a small repository summary from the current working directory, asks Codex for a read-only plan, and writes a JSONL session log.
- `run <skill-id>` loads the selected skill, executes it through Codex with minimal adapter wiring for future approval and diff hooks, and writes a JSONL session log.
- `dev [--] <command...>` runs an explicit local command, or falls back to `.repo-kit/config.yaml` `dev.command`, while teeing stdout and stderr into a JSONL session log.
- `auth status` reports whether `OPENAI_API_KEY` is present.

## Repository config

`repo-kit` optionally reads `.repo-kit/config.yaml` from the current working directory:

```yaml
dev:
    command: npm run dev
logs:
    dir: logs/repo-kit
project:
    name: my-service
```

- `dev.command`: default shell command for `repo-kit dev`
- `logs.dir`: optional directory override for JSONL session logs
- `project.name`: optional project label recorded in session metadata

## Skill Format

The loader prefers the normalized layout:

```text
skills/<skill-id>/
  skill.yaml
  prompt.md
  assets/
  templates/
  scripts/
```

Legacy content is still supported in v1:

- `SKILL.md`
- a single markdown file fallback for older one-off skills

Directories that only contain assets or templates are listed as invalid until they gain a runnable prompt definition.

## Publishing

The package name is scoped for GitHub Packages: `@dotnaos/repo-kit`.

```bash
git tag v1.0.0
git push origin v1.0.0
```

Publishing is handled by GitHub Actions only:

- push a version tag that matches `package.json`, for example `v1.0.0`
- `.github/workflows/publish.yml` installs dependencies, builds, and publishes to `https://npm.pkg.github.com`
- the workflow uses the standard `GITHUB_TOKEN` with `packages: write`

For a manual local dry run:

```bash
npm install
npm run build
npm publish --dry-run
```

## Project Layout

- `src/cli/`: CLI entrypoint and output formatting
- `src/core/`: skill loading, auth checks, repo context collection, shared types
- `src/adapters/codex-sdk/`: Codex SDK adapter
- `skills/`: bundled skill catalog
- `docs/`: usage and architecture notes

## v1 Limitations

- No agent backend other than Codex SDK
- No custom OAuth flow or ChatGPT login implementation
- No persistence layer beyond Codex thread/session handling and local JSONL session logs
- No approval or diff hooks yet, only the structure to add them later
