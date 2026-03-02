---
name: mobile-expo-codespaces
description: Setup and run Expo apps in GitHub Codespaces using tunnel for iPhone testing, and generate CI pipelines from reusable Jinja templates.
license: See repository license
---

## When to use

Use this skill when a repository contains an Expo app and you need a repeatable Codespaces workflow for:

- Running the app from GitHub Codespaces
- Testing on a physical iPhone with Expo Go via QR code
- Generating CI pipelines from reusable templates for build/submit/update tasks

## Workflow

1. Run environment checks first:

   ```bash
   bash ./scripts/doctor.sh
   ```

2. Start Expo in tunnel mode from Codespaces:

   ```bash
   bash ./scripts/start-tunnel.sh
   ```

   This runs `expo start --tunnel` (via `bunx` when Bun is used in the repo, otherwise `npx`).

3. Test on iPhone:

   - Open **Expo Go** on your iPhone.
   - Scan the QR code shown in the terminal.

4. If your app uses `expo-dev-client`, start with dev client tunnel mode:

   ```bash
   npx expo start --dev-client --tunnel
   ```

   If the repo uses Bun, the equivalent is:

   ```bash
   bunx expo start --dev-client --tunnel
   ```

5. Notes:

   - Tunnel mode is usually slower than LAN/Local, but works reliably for Codespaces + phone testing.
   - Port forwarding is optional, but tunnel is typically the simplest path for Expo Go on iPhone.

## Pipeline templates (Jinja, non-Markdown)

Copy a template file from `references/pipelines/` into your repository (usually `.github/workflows/`) and render it with your own values.

Included templates:

1. `references/pipelines/eas-build.yml.j2`
   - Generic EAS build pipeline.
   - Supports npm, yarn, pnpm, or bun.
   - Supports configurable branch filters, build profile, and platform.

2. `references/pipelines/eas-build-submit-update.yml.j2`
   - Generic multi-stage pipeline for EAS build, submit, and update.
   - Lets you enable/disable submit/update from template variables.
   - Useful as a more complete "all-the-things" mobile release pipeline.

Render the templates with any Jinja-compatible tool in your CI scaffolding flow.
