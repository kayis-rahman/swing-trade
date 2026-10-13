# Phase 2: Read-only edge with shadow comparison

Goal: Go serves all read endpoints identically, proven in shadow; production still answered by Java.

## Scope
Health, holidays and calendar reads, watchlist and stock reads, signals reads, positions and portfolio reads, settings reads, strategy config and type reads, job run reads, sentiment and news reads, report reads.

## Steps (per endpoint, TDD loop)
1. Ledger row; read Java controller, service, repository and tests.
2. Red: Go handler tests from Java tests plus goldens (status, body, headers, envelope).
3. Implement sqlc queries and handler; green.
4. Compare against goldens including edge cases (empty, paging, bad params).
5. Wire into the shadow comparator.
Then:
6. nginx mirrors production GETs to Go; Go responses are discarded; comparator logs diffs.
7. Triage every diff into bug, accepted difference (recorded with captain approval), or comparator noise.
8. Run for one full trading week.

## Gate G2 exit criteria
Zero unexplained diffs for one trading week; all accepted differences approved; Playwright suite green against Go reads in a test environment.
