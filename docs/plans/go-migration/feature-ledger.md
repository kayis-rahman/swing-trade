# Feature ledger

One row per feature. Columns: ID, Java module, Java classes, endpoints, schedulers, tables, metrics, settings, Java tests, Go module and file, red tests committed, green, parity evidence, status.

Status values: not-started, red, green, parity-pending, closed.

## Initial rows (to be expanded to endpoint, job, entity and metric level in Phase 0)
| ID | Area | Java module | Phase |
|---|---|---|---|
| F-API-HEALTH | Health endpoints and actuator aliases | api | 2 |
| F-API-SIGNALS | Signals reads and generation (incl. SSE) | api | 2, 6 |
| F-API-JOBRUNS | Job runs, orchestrator, reaper | api | 2, 6 |
| F-API-CANDSCAN | Candidate scans with SSE, pause, resume, cancel | api | 6 |
| F-API-POSITIONS | Positions, performance, risk summary | api | 2, 7 |
| F-API-PORTFOLIOS | Paper portfolios | api | 2 |
| F-API-SETTINGS | Settings, Pi lifecycle, connection tests | api | 2, 6 |
| F-API-STRATCFG | Strategy configs, types, report, promotion | api | 2, 6 |
| F-API-BACKTEST | Backtest endpoints and reports | api | 4 |
| F-API-INGEST | Ingestion, holidays, watchlist, data pull | api, data | 3 |
| F-API-SENTIMENT | Sentiment, news, pdf | api, llm | 5 |
| F-API-ADMIN | Admin, kill switch, data validate | api, broker | 6, 7 |
| F-API-AUTHFLOWS | Fyers and Upstox auth | api, data | 3 |
| F-API-GPUHUB | GPUHub endpoints | api, gpuhub | 5 |
| F-API-INTRADAY | Intraday paper sessions and scans | api, broker | 7 |
| F-API-SECURITY | API key, rate limit, CORS, error handler, trace id | api | 1 |
| F-DATA-CLIENTS | Yahoo, Upstox, Fyers clients, rate limiting | data | 3 |
| F-DATA-CALENDAR | NSE calendar, holidays, session grid | data | 3 |
| F-DATA-INGEST | EOD and intraday ingestion, reconciliation | data | 3 |
| F-DATA-ENTITIES | 35 entities, 37 repositories, 76 migrations | data | 2 to 8 |
| F-STRAT-IND | Indicators | strategy | 4 |
| F-STRAT-STRATS | Strategies and gates | strategy | 4 |
| F-STRAT-BT | Backtest, portfolio, intraday, walk-forward, DSR, costs | strategy | 4 |
| F-LLM-CLIENTS | Provider clients, selection, settings reload | llm | 5 |
| F-LLM-NEWS | 7 news sources, filtering | llm | 5 |
| F-LLM-SENT | Sentiment, synthesis, accuracy jobs | llm | 5 |
| F-LLM-PI | Pi lifecycle and pi-agent | llm | 5 |
| F-BRK-PAPER | Paper engine, orders, positions, snapshots, monitor | broker | 7 |
| F-BRK-RISK | Risk controls, kill switch, loss breaker | broker | 7 |
| F-BRK-KITE | Kite interface and refusing stub | broker | 7 |
| F-OBS | Logging, metrics, alerts, Pi thermal metrics | api, core | 1 |
| F-SCHED | 15 scheduled jobs and ownership flags | all | 3, 6 |
