# Go platform skeleton configuration

The Go executable starts on loopback by default and exposes infrastructure routes only. It does not load `.env` files itself. `bin/go-run` is the one-command local start path; use shell environment variables to override settings. The tracked `infra/env/.env.example` remains a template and must never contain real credentials.

| Environment variable | Existing Java property or Go setting | Default / behavior |
|---|---|---|
| `SERVER_PORT` | `server.port` | Go default `8081`; integer 1-65535 |
| `HTTP_HOST` | Go-only bind setting | `127.0.0.1`; set explicitly to `0.0.0.0` only for an authorized local container |
| `API_KEY_AUTH_ENABLED` | `app.security.api-key.enabled` | `false`; when enabled, an empty key is rejected at startup |
| `API_KEY` | `app.security.api-key` | Empty; required when API-key authentication is enabled; secret, never log it |
| `API_PUBLIC_RATE_LIMIT_PER_MINUTE` | `app.security.public-rate-limit-per-minute` | `100`; positive integer |
| `CORS_ALLOWED_ORIGINS` | `spring.web.cors.allowed-origins` | `http://localhost:3000,http://localhost:8080`; comma-separated exact origins |
| `DATABASE_URL` | Go-only direct URL equivalent of `spring.datasource.url` | Empty; if set, Go uses a pgx pool with `default_transaction_read_only=on` and rejects any target outside the authorized development host and port |
| `EOD_INGESTION_ENABLED` | Go-only job gate over Java's `data.eod.ingestion.enabled` | `false`; Go jobs stay off until the G8 switch even though Java defaults to `true`; starts the weekday 16:30 IST EOD loop when `DATABASE_URL` is set |
| `BENCHMARK_BACKFILL_ON_STARTUP` | Go-only job gate over Java's `benchmark.backfill.on-startup` | `false`; startup NIFTY50 backfill requires this plus `DATABASE_URL` |
| `BENCHMARK_BACKFILL_YEARS` | Go-only setting over Java's `benchmark.backfill.years` | `10`; positive integer |
| `FYERS_SYMBOL_MASTER_REFRESH_ENABLED` | Go-only job gate; Java has no equivalent and refreshes whenever Fyers is active | `false`; starts the weekday 08:45 IST refresh loop and the startup refresh-when-empty path when `DATABASE_URL` is set, and only when `selectedBroker` is `fyers` |
| `INTRADAY_INGESTION_ENABLED` | Go-only job gate over Java's `intraday.ingestion.enabled` | `false`; Go jobs stay off until the G8 switch even though Java defaults to `true`; starts the weekday 15:45 IST session-bar loop when `DATABASE_URL` is set |
| `INTRADAY_INGESTION_LOOKBACK_DAYS` | Go-only setting over Java's `intraday.ingestion.lookback-days` | `10`; any integer, negative values clamped to zero at run time like Java's `Math.max(0, lookbackDays)` |
| `SPRING_DATASOURCE_URL` | `spring.datasource.url` | Read at test runtime from the authorized, gitignored worktree env file; the verifier rejects non-dev targets |
| `SPRING_DATASOURCE_USERNAME` | `spring.datasource.username` | Read at test runtime only; never printed |
| `SPRING_DATASOURCE_PASSWORD` | `spring.datasource.password` | Read at test runtime only; never printed |

The verification script reads only those datasource settings at runtime from the authorized, mode-600, gitignored `infra/env/.env`; it never prints values or copies the file elsewhere. It refuses any target other than `192.168.0.100:5435` and fails closed if the file/settings are unavailable. Read tests use the read-only pool. Write-scope tests create a random `go_port_test_<hex>` schema, restrict `search_path` to it, and drop only that schema. They do not run migrations or access existing tables.

## Local commands

- Start the Go skeleton: `./bin/go-run` (default `127.0.0.1:8081`)
- Run unit, real-binary HTTP E2E, optional authorized development-database integration, race, vet, lint, vulnerability and architecture checks: `./bin/go-verify`

The HTTP E2E test builds and starts the actual `swingtrade-api` binary on an ephemeral loopback port and covers every exposed route, including admin authentication and CORS preflight. No container, stage, tunnel, or external service is used by that test.

The verification command enforces at least 80% statement coverage for every tested executable package. The `data/db` package reaches 80.0% through unit tests. The default `./bin/go-verify` path reads the approved datasource settings and tests only the dev database. Shell TCP reaches `192.168.0.100:5435`, but Go on macOS reports `connect: no route to host` before a session. Captain-authorized local verification is available with `SWINGTRADE_USE_LOCAL_TEST_DATABASE=1 ./bin/go-verify` when a throwaway Postgres listens only on `127.0.0.1:55433`; the test-only target check leaves the production pool guard unchanged. That full local verification passed, and no generated test schemas remained afterward. The dev database itself remains unverified from this process context. Empty module placeholders have no executable statements or coverage claim.
