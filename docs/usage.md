# Usage

## Requirements

- Node.js 18+
- `OPENAI_API_KEY` for agent-backed commands
- Run `plan` and `run` from inside a Git repository

## Commands

```bash
repo-kit skills list
repo-kit plan <skill-id>
repo-kit run <skill-id>
repo-kit dev [--] <command...>
repo-kit auth status
```

## Repository config

`repo-kit` optionally reads `.repo-kit/config.yaml` from the current working directory.

```yaml
dev:
    command: npm run dev
logs:
    dir: logs/repo-kit
project:
    name: my-service
```

- `dev.command`: shell command used by `repo-kit dev` when no explicit command is passed
- `logs.dir`: directory for JSONL session logs; defaults to `logs/repo-kit` under the current working directory
- `project.name`: optional project label recorded in session log metadata

### `repo-kit skills list`

- Scans first-level directories under `skills/`
- Prints the skill id, title when available, and whether the entry looks runnable
- Marks asset-only directories as invalid until they have a prompt definition

### `repo-kit plan <skill-id>`

- Loads the selected skill from the packaged `skills/` directory
- Collects a small repository summary from the current working directory
- Runs Codex in read-only planning mode
- Prints the intended plan without applying changes
- Writes a JSONL session log for the invocation

### `repo-kit run <skill-id>`

- Loads the selected skill from the packaged `skills/` directory
- Collects the same minimal repository context
- Runs Codex in execution mode with the skill directory exposed as an additional readable path
- Prints the final response, changed files, and executed commands when available
- Writes a JSONL session log for the invocation

### `repo-kit dev [--] <command...>`

- Runs an explicit command directly from the CLI arguments
- If no explicit command is provided, falls back to `.repo-kit/config.yaml` `dev.command`
- Streams stdout and stderr to the terminal while also recording line-oriented JSONL events
- Preserves the wrapped command's exit code via the `repo-kit` process exit status

### `repo-kit auth status`

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
npm install @dotnaos/repo-kit
```

Publish:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The publish workflow validates that the tag matches `package.json`, then runs `npm ci`, `npm run build`, and `npm publish` against GitHub Packages.
