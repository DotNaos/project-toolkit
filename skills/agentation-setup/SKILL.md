---
name: agentation-setup
description: Set up Agentation end to end in a local web project: read the current official docs first, install the toolbar, run the local Agentation server, configure MCP for Codex or another supported agent, and verify that the browser feedback loop actually works.
license: See repository license
---

# Agentation setup

## When to use

Use this skill when a project should support in-browser feedback through the Agentation toolbar and the agent should receive that feedback through MCP instead of copy-paste.

Typical cases:

- local web UI review and feedback loops
- Codex or another MCP-capable coding agent should see annotations directly
- the project needs a repeatable Agentation setup instead of one-off manual steps

## Source of truth

Before making changes, read the current official docs and treat them as authoritative if they differ from this skill:

- [Install](https://www.agentation.com/install)
- [MCP](https://www.agentation.com/mcp)

This skill is intentionally procedural. The docs above stay current; your implementation should follow them.

## Workflow

1. Inspect the target project first.
   - Identify the framework, package manager, dev server command, and root layout or shell where a dev-only toolbar can be mounted.
   - Confirm the app is a React-compatible surface. Agentation currently requires React 18+ and client-side DOM access.
   - If the repo is not a React app, Next.js app, or an Astro app that can host a React island, stop and explain the limitation instead of forcing an unsupported integration.

2. Install the minimum app-side dependencies.
   - Add `agentation` as a dev dependency.
   - If the framework needs extra React plumbing to host the toolbar, add only the minimal compatible pieces already aligned with the repo.
   - For Astro, prefer a tiny React wrapper component plus the framework's React integration instead of spreading Agentation logic through multiple files.

3. Mount the toolbar near the app root.
   - Keep the integration dev-only unless the user explicitly asks otherwise.
   - Use the local server endpoint when MCP sync is desired: `http://localhost:4747`.
   - Keep the wrapper small and focused. The toolbar component should exist for one reason only: render Agentation cleanly in development.
   - Preserve the repository's existing layout and visual structure; do not redesign the app just to add the toolbar.

4. Add a local server command to the project.
   - Prefer a project script such as `agentation:mcp` that starts `agentation-mcp server`.
   - Prefer local project dependencies or `npx` over undocumented global requirements.
   - Keep the command obvious so a human can start the server without reading code.

5. Configure MCP for the agent.
   - Prefer the official cross-agent command when generic setup is acceptable:
     - `npx add-mcp "npx -y agentation-mcp server"`
   - If the user specifically wants Codex configured and the `codex` CLI is available, prefer an explicit Codex entry:
     - `codex mcp add agentation -- npx -y agentation-mcp server --mcp-only --http-url http://localhost:4747`
   - After changing MCP config, note that already-running agent sessions may need a restart or a new thread before the new server becomes available.

6. Verify the setup instead of assuming it worked.
   - Run `npx agentation-mcp doctor` or the local package-manager equivalent.
   - Start the Agentation server.
   - Start the app dev server.
   - Use the [$agent-browser](/Users/oli/.agents/skills/agent-browser/SKILL.md) skill to open the app and confirm the toolbar is visible in the page.
   - If Codex was configured explicitly, verify with `codex mcp list` and `codex mcp get agentation` when available.
   - If the browser toolbar is present but MCP is not live in the current session yet, say so clearly and explain that a fresh agent session is required.

## Framework notes

- React / Vite:
  - Mount Agentation close to the root app component with a development guard.
- Next.js:
  - Use a client component near the root layout boundary. Keep the server/client split explicit.
- Astro:
  - Create a small React wrapper and render it with `client:only="react"` from a shared layout or top-level page shell.

## Guardrails

- Do not hardcode production behavior when the docs recommend a development-only setup.
- Do not assume the current agent session hot-reloads new MCP servers.
- Do not claim success before checking the page in a browser.
- Do not bury the setup in multiple unrelated files when one wrapper and one script are enough.
- Do not override existing framework conventions if a smaller integration path exists.

## Definition of done

- Agentation is installed in the project
- the toolbar is mounted in a sensible root location
- the toolbar only appears in development unless the user asked otherwise
- the project has an obvious command to run the Agentation server
- MCP is configured for the requested agent path
- doctor checks and browser verification were both run
- the final handoff explains any restart requirement for the agent session
