# Phase 8: Cutover and Java retirement

Goal: single switch to Go, a bounded rollback window, then retire Java.

## Steps
1. Freeze feature work on Java for the cutover window; confirm ledger is fully closed and traceability matrix has no unmapped tests.
2. Rehearse booting Java with `spring.jpa.hibernate.ddl-auto=validate` against a goose-migrated database copy and confirm it starts.
3. Take and verify a database backup; record goose baseline V76 as a no-op that marks the schema as applied, after verifying it equals the Phase 0 schema snapshot (migration ownership moves here, not before). `flyway_schema_history` is never dropped or edited.
4. Change nginx routes to Go; disable Java scheduling with the existing global `app.scheduling.enabled=false` and let Go take its per-job Postgres advisory locks; Java stays warm and read-only (all scheduling off).
5. Watch rollback triggers (rollback.md) continuously for the agreed window.
6. Run one full weekly cycle on Go (digests, monthly report where applicable).
7. Drill a rollback to Java once during the window.
8. After the window, remove Java modules, Gradle, Checkstyle, PMD, JaCoCo, Dockerfile.native and Java dev-stack paths; update `AGENTS.md`, `CLAUDE.md`, `README.md`, `.codex/RULES.md`, skills, `docs/`.
9. Update `docs/status.md` only after verification.

## Gate G8 exit criteria
Acceptance criteria in acceptance.md met; rollback drill done; captain signs off on retirement.
