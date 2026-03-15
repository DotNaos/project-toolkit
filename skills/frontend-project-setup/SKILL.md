---
name: frontend-project-setup
description: Scaffold a generic frontend project baseline with a copy-ready Jinja package.json template and common scripts.
license: See repository license
---

## When to use

Use this skill when you need to quickly bootstrap a frontend repository with a reusable `package.json` that can be generated for different stacks.

## Workflow

1. Copy `templates/package.json.j2` into your target project root.
2. Render it with your Jinja-compatible tooling using values for framework, package manager scripts, and optional quality tooling.
3. Install dependencies with your package manager of choice.
4. For local web apps, keep the dev server port configurable. Prefer `portless run ...` in the generated `dev` script or a toolkit `dev.router.mode: portless` wrapper instead of hardcoding `localhost:3000`.

## Template included

- `templates/package.json.j2`
  - Supports common frontend scripts (`dev`, `build`, `test`, `lint`, `format`).
  - The `dev` script should remain compatible with `PORT`/`HOST` injection so Portless can assign stable local URLs across worktrees.
  - Supports optional sections for `engines`, `lint-staged`, and `browserslist`.
  - Works as a generic baseline for React, Vue, Svelte, Next.js, and Vite-style projects by passing different values.
