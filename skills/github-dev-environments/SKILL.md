---
name: github-dev-environments
description: Use when preparing or migrating a repository for GitHub Codespaces or other dev-container-based GitHub developer VMs, especially when deciding whether the repo should use only .devcontainer or .devcontainer plus devbox.json.
---

## When to use

Use this skill when a repository needs a repeatable GitHub-hosted development environment and you need to:

- add a first `.devcontainer` setup
- migrate an ad hoc or outdated Codespaces setup
- decide whether `devbox.json` adds enough value to justify another config layer
- separate container concerns from project-tooling concerns

For this skill, "GitHub dev VMs" means GitHub Codespaces and similar dev-container-based GitHub environments.

## Workflow

1. Inspect the repository before choosing a shape.

   Check for:

   - `.devcontainer/devcontainer.json`
   - `devbox.json`
   - `Dockerfile`, `docker-compose.yml`, `compose.yaml`
   - `package.json`, `bun.lock`, `pnpm-lock.yaml`, `package-lock.json`
   - `go.mod`, `Cargo.toml`, `pyproject.toml`, `requirements.txt`
   - `.nvmrc`, `.tool-versions`, `mise.toml`
   - the actual local dev and bootstrap commands

2. Choose the target shape.

   Prefer:

   - **Devcontainer only** when the repo mostly needs editor integration, forwarded ports, and a small number of runtime/tool features.
   - **Devcontainer + Devbox** when the repo needs multiple native tools, stronger parity between local machines and Codespaces, or a reusable package/script layer outside the container definition.
   - **Existing Dockerfile or Compose integration** only when the repo already depends on that container layout or needs services that should stay containerized.

3. Render the templates from `templates/`.

   Use:

   - `templates/devcontainer.json.j2` for `.devcontainer/devcontainer.json`
   - `templates/devbox.json.j2` for `devbox.json` when Devbox is justified
   - `templates/post-create.sh.j2` for `.devcontainer/post-create.sh`

4. Keep responsibility boundaries clean.

   - `devcontainer.json` owns the container image/features, Codespaces machine requirements, forwarded ports, editor customizations, and lifecycle hooks.
   - `devbox.json` owns repo-scoped CLI packages, shell environment variables, and repeatable shell scripts.
   - Do not install the same runtime twice unless that duplication is deliberate and documented.

5. Validate the result.

   - Parse generated JSON before committing it.
   - Rebuild the dev container or Codespace.
   - Run the bootstrap path and the normal dev command.
   - Confirm forwarded ports and editor extensions/settings behave as expected.

## Configuration Rules

- Prefer a feature-based `devcontainer.json` over a custom Dockerfile unless features cannot express the needed setup cleanly.
- Put expensive, prebuild-friendly setup in `onCreateCommand` or `updateContentCommand`.
- Put repo bootstrap in `postCreateCommand`.
- Keep `postStartCommand` light; it runs often.
- Keep secrets out of committed config. Reference env files or secret managers instead.

## References

- Read `references/config-guide.md` when choosing template values and splitting responsibility between `devcontainer.json` and `devbox.json`.
- Read `references/migration-checklist.md` when converting an existing repo instead of starting fresh.
