# repo-kit

`repo-kit` is a reusable bootstrap + policy kit for GitHub repositories.

It provides:
- A Go CLI to initialize, sync, check, and update standardized repository files.
- A manifest + lock mechanism (`.repo-kit/config.yaml`, `.repo-kit/lock.json`) to reduce drift.
- Skills (in `assets/.github/skills/`) encapsulating reusable tooling, templates, and Terraform modules.

## Project Structure

- `cli/`: Go CLI (`init`, `sync`, `check`, `update`)
- `manifests/`: YAML manifests defining file mappings
- `assets/`: templates/workflows/skills copied into target repositories
  - `.github/skills/terraform-modules/`: Terraform modules for GitHub + GCP infrastructure
  - `.github/skills/mobile-expo-codespaces/`: Expo + Codespaces setup scripts and CI templates
  - `.github/skills/frontend-project-setup/`: Frontend project scaffold templates
- `docs/`: architecture and usage docs

## Quick Start

```bash
make build
./bin/repo-kit init --manifest default
./bin/repo-kit sync --kit-root .
./bin/repo-kit check
```
