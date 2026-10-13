# Universe-to-paper workflow

## Shipped behavior

- Manual and scheduled broad candidate scans now enumerate the same stored NSE symbol catalog (`fyers_symbol_master`). This removes the prior table-source drift; it does **not** establish that the stored catalog is a complete, current exchange security master.
- Candidate qualification is blocked when the latest stored daily candle is missing or older than the default four calendar-day maximum. The setting `candidate-scan.max-candle-age-days` accepts 0–14 days. The result remains visible as `STALE` with its latest available candle date and reason; stale rows cannot qualify or auto-promote.
- Qualified candidates continue to auto-promote into the active watchlist, as requested. Manual discovery no longer starts downstream analysis implicitly. The Candidate Explorer asks the operator to choose qualified candidates, active watchlist, or no analysis; the API (`POST /api/candidate-scans/{runId}/analysis`) accepts exactly `NONE`, `WATCHLIST`, or `QUALIFIED` (qualified, activated results) for a completed manual scan still in `NOT_REQUIRED` state. `NONE` is terminal (`SKIPPED`) and a scan's scope cannot be re-selected. There are no per-symbol scopes. Scheduled handoff continues to use qualified, activated symbols.
- Candidate Explorer and Intraday Paper remain separate review pages. Intraday paper session replay persists completed results and simulated records by run ID, and `GET /api/intraday/paper/sessions/{runId}` reloads the same result. Incomplete-data and kill-switch-blocked sessions are not persisted as completed paper ledgers.
- Intraday retrospective scan remains restricted to stored bars for the active NSE watchlist and fails closed above its existing configurable 20-symbol ceiling. No provider fetching or universe expansion was added.
- Paper session simulation continues using stored OHLCV and deterministic fills only. This change adds no broker, provider, or order-execution path.

## Decisions still needed

1. Confirm the authoritative universe/security-master source and inclusion/exclusion policy (instrument types, suspended symbols, symbol changes, and delistings). The implementation intentionally standardizes on the existing stored NSE catalog and does not invent a new feed or provider contract.
2. Replace or formally approve the four-calendar-day freshness default. It is a conservative interim gate, not an exchange-calendar-aware freshness SLO; long exchange holidays and scan time-of-day require a product/data policy.
3. Decide whether scheduled downstream analysis should remain automatic and whether operators need persisted attribution (identity, timestamp, rationale) for each selected scope. Current project authorization adds no identity model.
4. Define retention/export and exact audit requirements for persisted paper sessions. Current endpoint retains completed JSON records without a cleanup policy.
5. Do not enlarge intraday universe coverage until representative multi-session/calendar coverage, provider budget, and bounded batch throughput are measured. Current 20-symbol watchlist limit is preserved as the safe boundary; no broad intraday provider measurements were run here.

The worker-owned review board at `http://100.68.173.92:4387/session/3177c161c5ae72ba` could not be loaded in the isolated browser (navigation timed out); no board-only feedback was available to reconcile. Revisit that board before treating these defaults as final.

These choices should be resolved before claiming the stored catalog is the canonical market universe, promoting the freshness gate to a production SLO, or expanding intraday coverage.
