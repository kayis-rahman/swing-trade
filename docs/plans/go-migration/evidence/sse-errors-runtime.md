# Local SSE and error-response captures

Phase 0 step 4 runtime captures from 2026-10-17, using the real API jar at the current HEAD against an isolated throwaway PostgreSQL cluster. Scheduling and LLM were disabled. The Yahoo base URL was overridden to a dead loopback port so no external market-data provider was contacted. No stage service or database was accessed.

Raw responses, request/response headers, service logs, and the disposable database files are outside the worktree at:

`/Users/kayisrahman/Documents/workspace/firstmate/data/swing-trade-go-migration-impl-v1/sse-capture/`

## Captured cases

| Case | Result | Fixture |
|---|---|---|
| `POST /api/signals/generate-all/stream` | 200 `text/event-stream`; 30 progress events; terminal `COMPLETE` event says 0 signals and 14 skipped due to no candle data | `signals-sse.txt`, `signals-sse.headers` |
| `GET /api/candidate-scans/{runId}/stream` | 200 `text/event-stream`; connected to a 50-symbol run, captured snapshot/start/symbol/data-stage events and `RUN_CANCELLED` | `candidate-sse-live.txt`; response headers from the empty-run capture are in `candidate-sse.headers` |
| `POST /api/candidate-scans` during an active run | 409 JSON body: `{"error":"Candidate scan already running: <runId>"}`. The active run was cancelled immediately after the conflict response. | `candidate-conflict-rerun.json`, `candidate-conflict-rerun.headers`, `candidate-cancel.json` |
| `POST /api/intraday/paper/scans` with a seeded active watchlist larger than 20 | 422 with an empty body and `Content-Length: 0` | `intraday-422.headers`, `intraday-422.body` |

The candidate stream was exercised against 50 synthetic Fyers-symbol rows in the throwaway database. The run was cancelled after the conflict response, before any scan could finish. The local Yahoo endpoint was a closed loopback port; no external provider was contacted. The 422 fixture used the V1-seeded watchlist plus 21 synthetic active rows, all in the isolated database.

The signal-generation SSE completed normally on the seeded throwaway data. It generated no signals because there were no candles, so this is a stream-protocol and empty-data golden, not a populated-signal golden.

SHA-256:

- `signals-sse.txt`: `540cef8a5f13fc030793dae7aac8aec93cf55a3dd79be53f6b580b5c6af2d58e`
- `candidate-sse-live.txt`: `f85760e80c6eb424644adfa16f10816450a5ebbee2d4882e9cbbffbd9c775b7d`
- `candidate-conflict-rerun.json`: `777083689e4381ce2170b344e35b02edaed2f34045b23b1edde82047e66eb223`
- `intraday-422.body` (empty): `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

## Still outstanding

The third SSE endpoint, `POST /api/analysis/run-full`, was not invoked: it starts the full orchestration pipeline and may access external news or market-data sources. A safe bounded fixture for that pipeline remains outstanding. Numeric goldens for populated money paths and the remaining indicator, strategy, gate and backtest paths also remain outstanding.
