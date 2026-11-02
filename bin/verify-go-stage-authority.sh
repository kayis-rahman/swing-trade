#!/usr/bin/env bash
# Fail closed until an authorized verifier covers the shared pg-dev stage target.
set -euo pipefail

# The former drill targeted the standalone stage container and cannot establish
# backup/restore authority for swingtrade_stage on shared pg-dev. A replacement
# drill requires separate approval before any stage access.
echo "Stage authority unavailable: shared pg-dev backup/restore verification has not been approved." >&2
exit 1
