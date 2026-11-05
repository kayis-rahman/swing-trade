#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
echo "Scanning tracked test fixtures for obvious secret/token patterns..."
if grep -RInE 'AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{36,}|sk-[A-Za-z0-9]{20,}|api[_-]?key[[:space:]]*[:=][[:space:]]*["'"'"']?[^"'"'"'[:space:]]{8,}' --include='*.json' --include='*.yaml' --include='*.yml' --include='*.txt' . ; then
  echo "Potential secret found above." >&2
  exit 1
fi
echo "No obvious secrets found in tracked fixtures."
