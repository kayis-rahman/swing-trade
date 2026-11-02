# Scheduler soak report (Phase 6 step 6)

- Branch: fm/swing-trade-go-scheduler-soak-v1
- Scope: scheduler soak harness driving Go schedulers over simulated two weekly cycles with an injectable clock.
- Implementation: `go/data/scheduler/soak_harness.go` + tests in `go/data/scheduler/soak_harness_test.go`.
- Clock: injected `func() time.Time`, no direct `time.Now` in scheduler path.
- Database: disposable local PostgreSQL 16 only; never connected to shared pg-dev/stage/prod; temp data directory started and deleted by the operator.
- Advisory locks: schedulers that participate receive a per-job lock wrapper around `db.WithAdvisoryLock`; two harness instances sharing one database cannot both run the same job because the second `pg_try_advisory_lock` returns false.
- Discord: fake HTTP endpoint only; no production webhook is used.
- Notifications slice (PR 165): not yet merged in this attempt, so PR-165 schedulers (weekly sector digest, weekly strategy digest, monthly report) are coded against their documented interfaces (`Run(ctx)`, `Generate...`, constructor signatures) and will be re-run once PR 165 lands on main.
- Missed-run after restart: harness simulates restart by re-running cycles; idempotent seed data plus advisory locks prevent duplicate execution in the soak window.
- Not run in this environment: full multi-instance PostgreSQL soak against PR 165 schedulers, because those schedulers are not on main yet. EOD/intraday/fyers schedulers exercised via unit-level handlers in their own packages.
