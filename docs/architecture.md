# Architecture

`repo-kit` separates:

- **Materialized repository files** in `assets/`, selected by YAML manifests.
- **Remote repository/cloud policy** as Terraform modules in `assets/.github/skills/terraform-modules/assets/terraform/`.

CLI internals are split into config parsing (`internal/config`), filesystem operations (`internal/filesystem`), syncing (`internal/syncer`), lock management (`internal/lock`), and drift detection (`internal/checker`).
