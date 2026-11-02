# Scheduler soak report (Phase 6 step 6)

- Branch: fm/swing-trade-go-scheduler-soak-v1
- Scope: scheduler soak harness driving Go schedulers over simulated two weekly cycles with an injectable clock.
- Implementation: `go/data/scheduler/soak_harness.go` + tests in `go/data/scheduler/soak_harness_test.go`.
- Clock: injected `func() time.Time`, no direct `time.Now` in scheduler path.
- Database: disposable local PostgreSQL 16 only; never connected to shared pg-dev/stage/prod; temp data directory started and deleted by the operator.
- Advisory locks: not asserted by the harness itself; lock behavior is provided by the per-job `db.WithAdvisoryLock` wrappers used in `cmd/swingtrade-api` and is covered by job-level tests, not this harness.
- Discord: product delivery is Discord only; this harness does not call a notification endpoint.
- Notifications slice (PR 165): not yet merged in this attempt, so PR-165 schedulers (weekly sector digest, weekly strategy digest, monthly report) are coded against their documented interfaces (`Run(ctx)`, `Generate...`, constructor signatures) and will be re-run once PR 165 lands on main.
- Missed-run after restart: not simulated by the harness; restart/idempotency coverage relies on job-level idempotent writes and per-job advisory locks.
- Not run in this environment: full multi-instance PostgreSQL soak against PR 165 schedulers, because those schedulers are not on main yet. EOD/intraday/fyers schedulers exercised via unit-level handlers in their own packages.
