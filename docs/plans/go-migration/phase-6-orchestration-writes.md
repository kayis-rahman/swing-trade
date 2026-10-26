# Phase 6: Orchestration, signals, SSE and write APIs

Goal: the stateful pipeline and all write endpoints in Go, still not serving production.

## Scope
Job orchestrator (runs, stages, cancel and interrupt, orphan reaper, summary and notifier), signal pipeline and arbiter, candidate scans with pause/resume/cancel, the three SSE streams, settings writes, strategy config writes with `lock_version`, watchlist writes, admin endpoints (symbols, data validate), Discord and Telegram notifications, remaining schedulers.

## Steps
1. Ledger rows per stage, endpoint and scheduler.
2. Red tests from Java: orchestrator state machine, cancellation, reaper, arbitration, SSE event order and heartbeat, optimistic-lock conflicts.
3. Implement; green.
4. End-to-end runs on a scratch database comparing run summaries, stage details and persisted signals with Java.
5. Run the Playwright suite unchanged against Go on the scratch stack.
6. Scheduler soak: all Go schedulers against scratch data for two weekly cycles (digests, monthly report).

## Gate G6 exit criteria
Run outputs equal; SSE streams equal; Playwright green; two weekly cycles without duplicate or missed jobs.
