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
8. Replay a deterministic fixture corpus through both the legacy path and Go path; require zero unexplained diffs, and name and justify every tolerated difference in the parity report.
9. Run the live shadow comparator for one full trading week before release. This soak is a release-readiness gate, not a merge gate; do not represent deterministic replay as satisfying the live-week requirement.

## Gate G2 exit criteria
Zero unexplained diffs across the deterministic replay corpus; all tolerated differences named and justified; Playwright suite green against Go reads in a test environment. The one-trading-week live shadow soak remains outstanding until release readiness.
