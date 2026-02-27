# repo-kit

`repo-kit` is a reusable bootstrap + policy kit for GitHub repositories.

It provides:
- A Go CLI to initialize, sync, check, and update standardized repository files.
- A manifest + lock mechanism (`.repo-kit/config.yaml`, `.repo-kit/lock.json`) to reduce drift.
- Terraform examples for GitHub and GCP remote configuration.

## Project Structure

- `cli/`: Go CLI (`init`, `sync`, `check`, `update`)
- `manifests/`: YAML manifests defining file mappings
- `assets/`: templates/workflows copied into target repositories
- `terraform/`: remote policy examples (GitHub + GCP)
- `docs/`: architecture and usage docs

## Quick Start

```bash
make build
./bin/repo-kit init --manifest default
./bin/repo-kit sync --kit-root .
./bin/repo-kit check
```
