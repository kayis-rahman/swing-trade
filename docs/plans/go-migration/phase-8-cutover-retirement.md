# Phase 8: Cutover and Java retirement

Goal: single switch to Go, a bounded rollback window, then retire Java.

## Steps
1. Freeze feature work on Java for the cutover window; confirm ledger is fully closed and traceability matrix has no unmapped tests.
2. Take and verify a database backup; baseline the schema at V76 for goose (migration ownership moves here, not before).
3. Change nginx routes to Go; set every `scheduler.<job>.owner` to go; Java stays warm and read-only (jobs off).
4. Watch rollback triggers (rollback.md) continuously for the agreed window.
5. Run one full weekly cycle on Go (digests, monthly report where applicable).
6. Drill a rollback to Java once during the window.
7. After the window, remove Java modules, Gradle, Checkstyle, PMD, JaCoCo, Dockerfile.native and Java dev-stack paths; update `AGENTS.md`, `CLAUDE.md`, `README.md`, `.codex/RULES.md`, skills, `docs/`.
8. Update `docs/status.md` only after verification.

## Gate G8 exit criteria
Acceptance criteria in acceptance.md met; rollback drill done; captain signs off on retirement.
