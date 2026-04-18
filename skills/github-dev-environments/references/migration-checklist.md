# Migration Checklist

Use this checklist when converting an existing repository to a cleaner GitHub Codespaces or GitHub dev-VM setup.

## Inventory

- Confirm the real bootstrap command.
- Confirm the real day-to-day dev command.
- List required runtimes, CLIs, and native packages.
- Check whether the repo already has Docker, Compose, `.devcontainer`, or `devbox.json`.

## Target shape

- Choose `devcontainer` only, or `devcontainer` plus `devbox`.
- Keep Dockerfile or Compose only if the repo already depends on it in a meaningful way.
- Decide who owns each runtime: Dev Container Feature, Dockerfile, or Devbox.

## File updates

- Add or replace `.devcontainer/devcontainer.json`.
- Add `.devcontainer/post-create.sh` if bootstrap needs a script wrapper.
- Add `devbox.json` only when it reduces drift instead of adding noise.

## Cleanup

- Remove duplicate installs across features, shell scripts, and Devbox.
- Remove stale commands that no longer match the repo's actual package manager.
- Preserve useful forwarded ports, editor extensions, and editor settings from the old config.

## Verification

- Parse the generated JSON files.
- Rebuild the dev container or Codespace.
- Run bootstrap successfully.
- Run the normal dev command successfully.
- Check that the intended port opens and is labeled correctly.
- Check that editor customizations still load.
