#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

if [[ "${CODESPACES:-}" != "true" ]]; then
  echo "Info: CODESPACES=true not detected. This skill is primarily intended for GitHub Codespaces."
fi

if ! command -v node >/dev/null 2>&1; then
  echo "Error: Node.js is required but not found. Install Node.js in your environment and retry."
  exit 1
fi

use_bun=false
if [[ -f "$REPO_ROOT/bun.lock" || -f "$REPO_ROOT/bun.lockb" ]]; then
  use_bun=true
fi

if [[ "$use_bun" == "true" ]]; then
  if ! command -v bunx >/dev/null 2>&1; then
    echo "Error: Bun lockfile detected, but bunx is not available. Install Bun and retry."
    exit 1
  fi
  if ! bunx expo --version >/dev/null 2>&1; then
    echo "Error: Could not run 'bunx expo --version'. Ensure Expo CLI is available in the project."
    echo "Next step: install dependencies (for example 'bun install') and retry."
    exit 1
  fi
  echo "OK: bunx expo is available."
else
  if ! command -v npm >/dev/null 2>&1; then
    echo "Error: npm is required but not found. Install npm/Node.js and retry."
    exit 1
  fi
  if ! npx expo --version >/dev/null 2>&1; then
    echo "Error: Could not run 'npx expo --version'. Ensure Expo CLI is available in the project."
    echo "Next step: install dependencies (for example 'npm install') and retry."
    exit 1
  fi
  echo "OK: npx expo is available."
fi

echo "Doctor check passed."
