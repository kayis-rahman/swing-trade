#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

if [ -z "${LAYA_API_KEY:-}" ]; then
  echo "LAYA_API_KEY is not set. Export it before starting the service." >&2
  exit 1
fi

# First run downloads weights from Hugging Face; subsequent runs use the
# local cache and stay fully offline.
if [ -d "$HOME/.cache/huggingface/hub" ] && [ -n "$(ls -A "$HOME/.cache/huggingface/hub" 2>/dev/null)" ]; then
  export HF_HUB_OFFLINE=1
fi

exec uvicorn app:app --host 0.0.0.0 --port 8000
