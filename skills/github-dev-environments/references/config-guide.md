# GitHub Dev Environment Config Guide

Use this guide when filling the templates for GitHub Codespaces or another dev-container-backed GitHub setup.

## Recommended file split

- `.devcontainer/devcontainer.json`: container base image, dev container features, machine size, ports, editor customizations, lifecycle hooks.
- `.devcontainer/post-create.sh`: repo bootstrap that should run after the workspace is created.
- `devbox.json`: optional project-level packages, env, and scripts that should stay portable outside Codespaces too.

## Choose the shape first

### Use `devcontainer.json` only

Use this when:

- the repo only needs one main runtime plus a few editor tools
- Dev Container Features already cover the stack cleanly
- the local team does not need Devbox outside Codespaces

### Add `devbox.json`

Use this when:

- the repo needs several native tools or CLIs
- local laptops and Codespaces should share the same package list
- you want stable repo-scoped scripts like `devbox run bootstrap` or `devbox run dev`

### Keep Dockerfile or Compose in the loop

Use this when:

- the repo already has a meaningful build/runtime container
- local and CI workflows already depend on that Docker layout
- supporting services belong in containers rather than shell tools

## `devcontainer.json.j2` inputs

- `name`: friendly environment name
- `image`: base container image; keep this simple unless the repo already needs Dockerfile control
- `workspace_folder`: default working folder inside the container
- `remote_user`: container user
- `features`: Dev Container Features map for runtimes and tools
- `forward_ports`: ports to auto-forward
- `ports_attributes`: per-port labels or behavior
- `container_env`: container-scoped environment variables
- `vscode_extensions`: editor extensions to install in the container
- `vscode_settings`: default editor settings inside the container
- `host_requirements`: minimum Codespaces machine shape
- `on_create_command`: one-time setup before the repo is fully ready
- `update_content_command`: prebuild-friendly dependency preparation
- `post_create_command`: repo bootstrap after create/rebuild
- `post_start_command`: light command that can run at every start

## `devbox.json.j2` inputs

- `packages`: repo-scoped packages managed by Devbox
- `env`: environment variables exposed by Devbox
- `include`: optional Devbox plugins
- `init_hook`: quick shell setup steps
- `scripts`: named commands for `devbox run`

If `post-create.sh` runs `devbox run bootstrap`, make sure `scripts.bootstrap` exists.

## `post-create.sh.j2` inputs

- `use_devbox`: enables the Devbox path
- `devbox_install_command`: installs Devbox if the base image does not already provide it
- `devbox_bootstrap_command`: usually `devbox run bootstrap`
- `bootstrap_command`: non-Devbox fallback bootstrap command
- `post_create_steps`: optional extra shell lines

## Split responsibilities, do not duplicate them

- If a runtime comes from a Dev Container Feature, do not install the same runtime again via Devbox unless you intentionally want Devbox to own the version.
- If Devbox owns repo scripts, keep `postCreateCommand` thin and point it at the script entrypoint.
- If the repo already has a package manager lockfile, let the bootstrap path use that package manager directly instead of inventing another layer.

## Good defaults

- Start with a simple image-based `devcontainer.json`.
- Add only the features the repo truly needs.
- Use `host_requirements` only when the default Codespaces machine is predictably too small.
- Keep `postStartCommand` minimal to avoid slow starts.
