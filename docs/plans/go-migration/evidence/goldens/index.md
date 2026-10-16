# Golden corpus index

Capture tag: `local-throwaway`. Base: `http://127.0.0.1:8080`.

| case | method | path | status | expected | note |
|---|---|---|---|---|---|
| health | GET | `/api/health` | 200 | 200 | HealthController aggregate health |
| health-database | GET | `/api/health/database` | 200 | 200 | DB health probe |
| health-details | GET | `/api/health/details` | 200 | 200 | health details |
| health-full | GET | `/api/health/full` | 200 | 200 | full health (nginx rewrites this in stage) |
| health-llm | GET | `/api/health/llm` | 200 | 200 | llm health |
| health-market-data | GET | `/api/health/market-data` | 200 | 200 | market data provider health |
| actuator-health | GET | `/actuator/health` | 200 | 200 | actuator health, unauthenticated |
| actuator-prometheus | GET | `/actuator/prometheus` | 200 | 200 | Prometheus scrape endpoint |
| admin-health | GET | `/api/admin/health` | 200 | 200 | admin health aggregate |
| admin-data-status | GET | `/api/admin/data/status?symbol=RELIANCE&days=30` | 500 | 500 | admin ingestion status (symbol and days are required) |
| admin-symbols-status | GET | `/api/admin/symbols/status` | 200 | 200 | symbol universe status |
| admin-kill-switch-status | GET | `/api/admin/kill-switch/status` | 200 | 200 | kill switch state read |
| settings | GET | `/api/settings` | 200 | 200 | settings bundle |
| settings-llm | GET | `/api/settings/llm` | 200 | 200 | llm settings |
| settings-trading | GET | `/api/settings/trading` | 200 | 200 | trading settings |
| settings-scanning | GET | `/api/settings/scanning` | 200 | 200 | scanning settings |
| settings-discord | GET | `/api/settings/discord` | 200 | 200 | discord settings |
| settings-gpuhub | GET | `/api/settings/gpuhub` | 200 | 200 | gpuhub settings |
| settings-pi-status | GET | `/api/settings/pi/status` | 200 | 200 | pi ssh status |
| signals-latest | GET | `/api/signals/latest` | 200 | 200 | latest signals list |
| signals-high-confidence | GET | `/api/signals/high-confidence` | 200 | 200 | high confidence signals |
| signals-date-range | GET | `/api/signals/date-range?startDate=2026-01-01&endDate=2026-01-31` | 200 | 200 | date-range query params |
| signals-gate-effectiveness | GET | `/api/signals/gate-effectiveness?from=2026-01-01&to=2026-01-31` | 200 | 200 | gate effectiveness report (from/to required) |
| signals-symbol | GET | `/api/signals/symbol/RELIANCE` | 200 | 200 | per-symbol signals, empty symbol |
| signals-combined | GET | `/api/signals/combined/RELIANCE` | 200 | 200 | captured 200 degradation response with keyword fallback when the LLM is unreachable; live news content is network-dependent |
| signals-analysis | GET | `/api/signals/analysis/RELIANCE` | 200 | 200 | stored analysis for a symbol |
| positions | GET | `/api/positions` | 200 | 200 | open positions list |
| positions-closed | GET | `/api/positions/closed` | 200 | 200 | closed positions list |
| positions-performance | GET | `/api/positions/performance` | 200 | 200 | performance summary |
| positions-stats | GET | `/api/positions/stats` | 200 | 200 | position statistics |
| positions-sector-allocation | GET | `/api/positions/sector-allocation` | 200 | 200 | sector allocation |
| positions-risk-summary | GET | `/api/positions/risk-summary` | 200 | 200 | risk summary |
| positions-symbol | GET | `/api/positions/symbol/RELIANCE` | 200 | 200 | positions for a symbol |
| watchlist | GET | `/api/watchlist` | 200 | 200 | watchlist list |
| watchlist-all | GET | `/api/watchlist/all` | 200 | 200 | full watchlist read |
| strategy-types | GET | `/api/strategy-types` | 200 | 200 | strategy type registry |
| strategy-configs | GET | `/api/strategy-configs` | 200 | 200 | strategy configuration list |
| strategy-report | GET | `/api/strategy-report` | 200 | 200 | strategy report |
| backtest-reports | GET | `/api/backtest/reports` | 200 | 200 | backtest report index |
| backtest-strategies | GET | `/api/backtest/strategies` | 200 | 200 | backtest strategies |
| sentiment-accuracy | GET | `/api/sentiment/accuracy` | 200 | 200 | sentiment accuracy |
| sentiment-accuracy-summary | GET | `/api/sentiment/accuracy/summary` | 200 | 200 | sentiment accuracy summary |
| sentiment-symbol-latest | GET | `/api/sentiment/RELIANCE/latest` | 200 | 200 | latest sentiment for a symbol |
| sentiment-symbol-history | GET | `/api/sentiment/RELIANCE/history` | 200 | 200 | sentiment history |
| sentiment-evaluate-status | GET | `/api/sentiment/evaluate/status` | 200 | 200 | evaluation job status |
| synthesis-evaluations-summary | GET | `/api/synthesis/evaluations/summary` | 200 | 200 | synthesis evaluation summary |
| holidays | GET | `/api/holidays` | 200 | 200 | NSE holiday list |
| holidays-today | GET | `/api/holidays/today` | 200 | 200 | today's holiday state |
| data-status | GET | `/api/data/status` | 200 | 200 | ingestion status |
| data-export | GET | `/api/data/export?symbol=RELIANCE&fromDate=2026-01-01&toDate=2026-01-31` | 200 | 200 | data export (CSV) |
| data-pull-progress | GET | `/api/data/pull/progress` | 200 | 200 | pull progress |
| ingestion-status | GET | `/api/ingestion/status` | 200 | 200 | ingestion status |
| ingestion-reconcile-out-of-range | GET | `/api/ingestion/reconcile?from=2026-01-01&to=2026-01-31` | 400 | 400 | reconcile with a 31-day window: range guard |
| ingestion-reconcile | GET | `/api/ingestion/reconcile?from=2026-01-05&to=2026-01-10` | 200 | 200 | reconcile report inside the allowed window |
| job-runs | GET | `/api/job/runs` | 200 | 200 | job run list |
| candidate-scans | GET | `/api/candidate-scans` | 200 | 200 | candidate scan list |
| candidate-scan-settings | GET | `/api/candidate-scans/settings` | 200 | 200 | candidate scan settings |
| paper-portfolios | GET | `/api/paper-portfolios` | 200 | 200 | paper portfolio list |
| signal-selections | GET | `/api/signal-selections` | 200 | 200 | operator signal selections |
| strategy-config-variant | GET | `/api/strategy-configs/unknown-variant` | 400 | 400 | single config read, unknown id: 400 not 404 |
| symbol-strategy-matrix | GET | `/api/symbols/RELIANCE/strategy-matrix` | 200 | 200 | per-symbol strategy matrix |
| fyers-status | GET | `/api/fyers/status` | 200 | 200 | fyers auth status |
| news-symbol-latest | GET | `/api/news/RELIANCE/latest` | 200 | 200 | news for a symbol |
| pdf-symbol-latest | GET | `/api/pdf/RELIANCE/latest` | 200 | 200 | extracted PDF snapshot |
| not-found-signals-symbol | GET | `/api/signals/symbol/UNKNOWN` | 200 | 200 | per-symbol read for a symbol with no rows (empty list, not 404) |
| not-found-unknown-collection | GET | `/api/definitely-not-a-real-path` | 404 | 404 | unmapped path: Spring's default error body |
| validation-watchlist-empty-body | POST | `/api/watchlist` | 400 | 400 | POST with an empty JSON object: bean-validation errors |
| conflict-watchlist-missing-symbol | POST | `/api/watchlist` | 400 | 400 | POST with a body that omits the required symbol |
| validation-signals-generate-empty | POST | `/api/signals/generate` | 400 | 400 | POST generate with an empty body |
| reserved-segment-signals-delete-generate | DELETE | `/api/signals/generate` | 200 | 200 | DELETE on 'generate': answered by the {symbol} handler, so 'generate' is NOT a reserved segment |
| reserved-segment-signals-delete-latest | DELETE | `/api/signals/latest` | 400 | 400 | DELETE on the reserved segment 'latest': bare 400 {"error": ...} body, not the ApiResponse envelope |
| intraday-paper-session-not-found | GET | `/api/intraday/paper/sessions/gnhf-unknown-run` | 404 | 404 | unknown runId: Spring's default 404 error body |
| intraday-paper-session-empty-body | POST | `/api/intraday/paper/sessions` | 400 | 400 | POST with an empty JSON object: bean-validation errors, fieldErrors still serialized as [] |
| intraday-paper-session-no-bars | POST | `/api/intraday/paper/sessions` | 200 | 200 | paper replay with no stored bars: 200 with status INCOMPLETE_DATA and no session row persisted |
| intraday-paper-session-non-intraday-strategy | POST | `/api/intraday/paper/sessions` | 400 | 400 | non-INTRADAY_ strategy: empty-bodied 400, no bars loaded and no execution |

The four `intraday-paper-*` rows were captured on 2026-10-15 against a *fresh* throwaway cluster (same V1 seed, tag `local-throwaway-fixed`) after the Jackson 3 fix in golden-corpus.md §1; every other row is tag `local-throwaway`. Use the capture script's `--only` flag for a partial re-capture, since the public rate window is one bucket per minute.

## Observed formatting facts

```json
{
 "health_timestamp_example": "2026-10-15T07:44:09.765538",
 "health_headers_sample": {
  "Content-Type": "application/json",
  "Transfer-Encoding": "chunked",
  "Date": "Thu, 15 Oct 2026 02:14:09 GMT"
 }
}
```

SSE endpoints (capture separately, they do not terminate):

* POST /api/signals/generate-all/stream
  * GET  /api/candidate-scans/{runId}/stream
  * POST /api/analysis/run-full

Capture the first N events with `curl -N` and store the raw text as the golden.
