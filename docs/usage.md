# Usage

## Requirements

- Node.js 18+
- `OPENAI_API_KEY` for agent-backed commands
- Run `plan` and `run` from inside a Git repository

## Commands

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

## Repository config

`project-toolkit` optionally reads `.project-toolkit/config.yaml` from the current working directory.

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

- `dev.command`: shell command used by `pkit dev` when no explicit command is passed
- `logs.dir`: directory for JSONL session logs; defaults to `logs/project-toolkit` under the current working directory
- `project.name`: optional project label recorded in session log metadata
- `workspace.baseFile`: stable in-repo workspace template used as the source for future generated workspaces
- `shared`: flat list of shared gitignored paths; `source` and `target` default to `path`, while `include` / `exclude` filter by worktree name

### `pkit project init [--force]`

- Creates `.project-toolkit/config.yaml` when missing
- Creates `.project-toolkit/base.code-workspace` when missing
- With `--force`, rewrites both scaffold files
- Seeds the future worktree/shared-link foundation without requiring manual file creation

### `pkit project workspace generate [--name <workspace>] [--root <dir>] [--output <file>]`

- Loads `.project-toolkit/base.code-workspace` or the configured `workspace.baseFile`
- Replaces the entire `folders` section with exactly one entry pointing at the chosen root
- Writes the generated workspace file outside the repo by default under `~/.project-toolkit/projects/<project-key>/workspaces/`
- Applies matching `shared` entries as symlinks from the main repo into the chosen root
- Uses the workspace name for `shared.include` / `shared.exclude` filtering

### `pkit project worktree create <name> [--branch <branch>] [--base <ref>] [--workspace <workspace>] [--output <file>]`

- Creates a managed worktree under `~/.project-toolkit/projects/<project-key>/worktrees/<name>`
- Creates a new branch with the same name by default, or reuses an existing local branch when present
- Uses `--base <ref>` for new branches, defaulting to the repository's current `HEAD`
- Generates a matching workspace file and applies shared-file links into the new worktree
- Keeps the main repository as the shared-file source in v1

### `pkit skills list`

- Scans first-level directories under `skills/`
- Prints the skill id, title when available, and whether the entry looks runnable
- Marks asset-only directories as invalid until they have a prompt definition

### `pkit plan <skill-id>`

- Loads the selected skill from the packaged `skills/` directory
- Collects a small repository summary from the current working directory
- Runs Codex in read-only planning mode
- Prints the intended plan without applying changes
- Writes a JSONL session log for the invocation

### `pkit run <skill-id>`

- Loads the selected skill from the packaged `skills/` directory
- Collects the same minimal repository context
- Runs Codex in execution mode with the skill directory exposed as an additional readable path
- Prints the final response, changed files, and executed commands when available
- Writes a JSONL session log for the invocation

### `pkit dev [--] <command...>`

- Runs an explicit command directly from the CLI arguments
- If no explicit command is provided, falls back to `.project-toolkit/config.yaml` `dev.command`
- Streams stdout and stderr to the terminal while also recording line-oriented JSONL events
- Preserves the wrapped command's exit code via the `pkit` process exit status

### `pkit auth status`

- Checks `OPENAI_API_KEY`
- Reports whether v1 authentication is available

## Local Development

```bash
npm install
npm run build
node dist/cli/index.js skills list
node dist/cli/index.js dev -- npm run build
```

## GitHub Packages

This package is configured for GitHub Packages only.

Install:

```bash
npm config set @dotnaos:registry https://npm.pkg.github.com
export NODE_AUTH_TOKEN=YOUR_GITHUB_TOKEN
npm install @dotnaos/project-toolkit
```

Publish:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The publish workflow validates that the tag matches `package.json`, then runs `npm ci`, `npm run build`, and `npm publish` against GitHub Packages.
