# Phase 3: Market data and ingestion

Goal: provider clients, calendar logic and ingestion jobs in Go with identical stored data.

## Scope
Yahoo, Upstox and Fyers HTTP clients, rate limiting, candle validation, NSE calendar and holidays, EOD and intraday ingestion, reconciliation, symbol master refresh, backfill and range endpoints. The unused Fyers SDK jar is not ported.

## Steps
1. Ledger rows per client, scheduler and endpoint.
2. Red tests from Java: candle validator, rate limiter, market calendar, session grid (09:15 to 15:15 IST, 25 bars), holiday service, reconciliation.
3. Recorded provider fixtures (payloads captured once, secrets removed).
4. Implement clients, validator, calendar, ingestion; green.
5. Run Go ingestion into a scratch database against the same sessions as Java; compare rows exactly (OHLCV, adjustments, rejects, audit rows).
6. Scheduler ownership: `scheduler.<job>.owner = java|go|off` setting plus Postgres advisory lock per job; during this phase Go jobs write only to scratch databases.
7. Failure drills: provider timeout, rate limit, malformed payload, partial day.

## Gate G3 exit criteria
Row-for-row equality over the agreed session window; failure behavior matches; no duplicate job runs in the ownership test.
