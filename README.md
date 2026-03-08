# project-toolkit

`project-toolkit` is a Node.js / TypeScript CLI for project operations, repository skills, and agent workflows through the official Codex SDK.

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
npm install @dotnaos/project-toolkit
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
pkit skills list
pkit project init [--force]
pkit project workspace generate [--name <workspace>] [--root <dir>] [--output <file>]
pkit project worktree create <name> [--branch <branch>] [--base <ref>] [--workspace <workspace>] [--output <file>]
pkit plan <skill-id>
pkit run <skill-id>
pkit dev [--] <command...>
pkit auth status
```

Command behavior in v1:

- `skills list` discovers first-level directories under `skills/` and reports whether each one looks runnable.
- `project init [--force]` scaffolds `.project-toolkit/config.yaml` plus `.project-toolkit/base.code-workspace` as the starting point for generated workspaces and shared-link config.
- `project workspace generate ...` loads the base workspace file, replaces its `folders` section for the chosen root, writes the generated `.code-workspace` file outside the repo by default, and applies matching shared-file symlinks.
- `project worktree create ...` creates a managed Git worktree under the toolkit state directory, creates or reuses a branch, and generates a matching workspace file for that worktree.
- `plan <skill-id>` loads the selected skill, collects a small repository summary from the current working directory, asks Codex for a read-only plan, and writes a JSONL session log.
- `run <skill-id>` loads the selected skill, executes it through Codex with minimal adapter wiring for future approval and diff hooks, and writes a JSONL session log.
- `dev [--] <command...>` runs an explicit local command, or falls back to `.project-toolkit/config.yaml` `dev.command`, while teeing stdout and stderr into a JSONL session log.
- `auth status` reports whether `OPENAI_API_KEY` is present.

## Repository config

`project-toolkit` optionally reads `.project-toolkit/config.yaml` from the current working directory:

```yaml
dev:
    command: npm run dev
logs:
    dir: logs/project-toolkit
project:
    name: my-service
workspace:
    baseFile: .project-toolkit/base.code-workspace
shared:
    - path: .env
```

- `dev.command`: default shell command for `pkit dev`
- `logs.dir`: optional directory override for JSONL session logs
- `project.name`: optional project label recorded in session metadata
- `workspace.baseFile`: stable workspace template inside the repo; future workspace generation replaces the active `folders` section from this base file
- `shared`: flat list of shared gitignored paths for future worktree linking; `source`/`target` default to `path`, while `include`/`exclude` scope entries by worktree name

Generated workspaces default to:

- `~/.project-toolkit/projects/<project-key>/workspaces/<workspace>.code-workspace`

The command accepts:

- `--name <workspace>`: logical workspace name, also used for shared-link include/exclude matching
- `--root <dir>`: target repository/worktree root that becomes the only generated `folders` entry
- `--output <file>`: override the generated workspace file path

Managed worktrees default to:

- `~/.project-toolkit/projects/<project-key>/worktrees/<name>`

`project worktree create` accepts:

- `<name>`: logical worktree identifier and default branch/workspace name
- `--branch <branch>`: override the Git branch name
- `--base <ref>`: base ref for new branches; ignored when the branch already exists
- `--workspace <workspace>`: override the generated workspace name
- `--output <file>`: override the generated workspace file path

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

The package name is scoped for GitHub Packages: `@dotnaos/project-toolkit`.

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
