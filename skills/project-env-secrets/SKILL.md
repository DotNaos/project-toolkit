---
name: project-env-secrets
description: Configure project-level environment loading with a split between committed config schema, local non-secret overrides, and 1Password-backed secrets. Use when a repository should support `.env.example`, `.env.local`, and `.env.op` (or equivalent), must actually load multiple env files in the correct order at runtime, and should keep project-specific values out of home-directory dotfiles.
license: See repository license
---

## When to use

Use this skill when you need to make a repository's environment setup repeatable and explicit:

- keep variable names and defaults documented in the repo
- keep non-secret local config separate from secrets
- keep secrets in 1Password and materialize them into `.env.op` or inject them at run time
- ensure the application, scripts, and local tooling really load both files in the right order

## Target model

Prefer this split unless the framework already has a better built-in convention:

1. `.env.example`
   - committed
   - documents all variable names and safe defaults/placeholders

2. `.env.local`
   - gitignored
   - local non-secret overrides

3. `.env.op`
   - gitignored
   - secrets from 1Password
   - not edited manually

Load order:

1. `.env.local`
2. `.env.op`
3. process environment as final override

Secrets should override local config when the same key exists.

## Workflow

1. Inspect the repository's actual runtime and startup path before changing file names.
   - Check application bootstrap, package scripts, Docker compose files, test runners, dev containers, and CI helpers.
   - Do not assume the project already loads multiple env files just because the files exist.

2. Keep project-specific variables in the project.
   - Do not put project env vars into home-directory dotfiles.
   - Only global user/service tokens belong in user dotfiles.

3. Implement file loading in the project, not just the file layout.
   - Add or update the real loader so `.env.local` and `.env.op` are consumed in the correct order.
   - Reuse framework-native conventions when available.
   - If the framework does not support this directly, add the smallest explicit loader possible.

4. Update repository hygiene.
   - Commit `.env.example`.
   - Ignore `.env.local` and `.env.op`.
   - Add or update setup documentation so another developer can reproduce the flow.

5. Wire 1Password around secrets only.
   - Treat 1Password as the source of truth for secret values.
   - Keep ordinary non-secret config in repo-visible files.
   - If the team uses 1Password Environments, import the keys from `.env.example`, populate the secrets there, and generate `.env.op`.

## Guardrails

- Do not move non-secret config into 1Password just for symmetry.
- Do not keep project-specific secrets in shared dotfiles.
- Do not introduce a second env system when the framework already has one that can be extended cleanly.
- Do not stop at documentation; make the runtime actually load the files.

## Implementation patterns

See `references/runtime-patterns.md` and choose the smallest pattern that matches the repository.

## Definition of done

- `.env.example` exists and reflects the expected keys
- `.env.local` and `.env.op` are gitignored
- the app, scripts, or compose setup really load both files in the intended order
- project docs explain how to create `.env.op` from 1Password and how to run locally
