# Runtime Patterns

Use the narrowest pattern that fits the repository instead of forcing the same implementation everywhere.

## Node.js services and CLIs

If the app already uses `dotenv`, load files explicitly in order:

```ts
import dotenv from "dotenv";

dotenv.config({ path: ".env.local", override: false });
dotenv.config({ path: ".env.op", override: true });
```

If startup happens through npm scripts, a preload entrypoint is often enough. Avoid scattering env loading across many files.

## Vite, Next.js, Astro and similar web stacks

Prefer framework-native env loading first. Many frameworks already load `.env.local`.

If `.env.op` is needed as an extra layer:

- add a tiny preload/bootstrap step for server-side processes, or
- use a framework-supported loader hook if one exists, or
- generate a file name that the framework already loads if that is cleaner for the project

Do not bolt on an extra dotenv stack if the framework already provides a clean mechanism.

## Docker Compose

Use multiple env files in declared order where supported, or a wrapper script that exports `.env.local` first and secrets second.

Keep secrets out of the compose file itself when possible.

## Python

If the project uses `python-dotenv`, load files explicitly:

```python
from dotenv import load_dotenv

load_dotenv(".env.local", override=False)
load_dotenv(".env.op", override=True)
```

Keep the load near the real app entrypoint.

## .NET / ASP.NET Core

Prefer native configuration layers first:

- `appsettings.json`
- `appsettings.Development.json`
- optional local untracked config such as `appsettings.Local.json`
- environment variables

If the repository still wants `.env.local` and `.env.op`, do not replace the native config stack. Instead, wire env-file loading in the local startup path or development scripts so those files become environment variables before the app starts.

## Shell / generic local tooling

If the project starts through shell scripts, load files explicitly in order:

```bash
set -a
[ -f .env.local ] && source .env.local
[ -f .env.op ] && source .env.op
set +a
```

This is acceptable for shell-compatible env files. Prefer simpler runtime-native loaders when the project language already has one.

## 1Password integration

Two common patterns:

1. Generate `.env.op`
   - import keys from `.env.example`
   - store secret values in 1Password
   - materialize `.env.op` locally

2. Run with 1Password directly
   - use `op run` for local commands
   - keep `.env.local` as the non-secret layer

Choose one primary workflow for the repository and document it. Avoid mixing multiple local startup paths unless the project truly needs them.
