---
name: terraform-modules
description: Reusable Terraform modules for GitHub repository settings and GCP infrastructure, including branch protection, repo configuration, Secret Manager, and Workload Identity.
license: See repository license
---

## When to use

Use this skill when you need to apply infrastructure-as-code for:

- Configuring GitHub repository settings (visibility, issue tracking, branch policies)
- Enforcing branch protection rules with required PR reviews
- Provisioning GCP Secret Manager secrets
- Setting up GCP Workload Identity pools for keyless authentication from GitHub Actions

## Modules

### `assets/terraform/github/repo_settings`

Creates or configures a GitHub repository with standard defaults:

- Private visibility
- Issues enabled
- Delete branch on merge enabled

**Variables**: `name` (string)

### `assets/terraform/github/branch_protection`

Enforces branch protection on a repository branch:

- Requires at least 1 approving PR review
- Configurable branch pattern (default: `main`)

**Variables**: `repository` (string), `pattern` (string, default `"main"`)

### `assets/terraform/gcp/secret_manager`

Creates a GCP Secret Manager secret with automatic replication.

**Variables**: `project_id` (string), `secret_id` (string)

### `assets/terraform/gcp/workload_identity`

Creates a GCP Workload Identity pool for use with GitHub Actions OIDC authentication.

**Variables**: `project_id` (string), `pool_id` (string)

## Workflow

1. Copy the relevant module directory into your infrastructure repository.

2. Reference the module from your root Terraform configuration:

   ```hcl
   module "repo" {
     source = "skills/terraform-modules/assets/terraform/github/repo_settings"
     name   = "my-repo"
   }

   module "branch_protection" {
     source     = "skills/terraform-modules/assets/terraform/github/branch_protection"
     repository = module.repo.name
   }
   ```

3. Initialize and apply:

   ```bash
   terraform init
   terraform apply
   ```

4. For GCP modules, ensure the `google` provider is authenticated (e.g. via `gcloud auth application-default login` or a service account key).
