#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

if [[ -f "$REPO_ROOT/bun.lock" || -f "$REPO_ROOT/bun.lockb" ]]; then
  CMD=(bunx expo start --tunnel)
else
  CMD=(npx expo start --tunnel)
fi

echo "Starting Expo development server in tunnel mode: ${CMD[*]}"
if ! "${CMD[@]}"; then
  echo
  echo "Expo tunnel start failed."
  echo "If the error mentions ngrok/tunnel dependencies, add @expo/ngrok to your project and retry:"
  echo "  npm install --save-dev @expo/ngrok"
  echo "  # or, for Bun projects"
  echo "  bun add -d @expo/ngrok"
  exit 1
fi
