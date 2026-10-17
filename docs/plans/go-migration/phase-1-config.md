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
| `DATABASE_URL` | Go-only direct URL equivalent of `spring.datasource.url` | Empty; if set, Go uses a pgx pool with `default_transaction_read_only=on` and rejects any target other than the authorized development DB at `192.168.0.100:5435/swingtrade_db` |
| `DB_HOST`, `DB_PORT`, `DB_NAME` | Existing Java datasource URL placeholders | Used by the verification script only after enforcing the authorized development target |
| `DB_USER`, `DB_PASSWORD` | Existing Java datasource username/password placeholders | Read from ignored `infra/env/.env` for that test connection; never printed |

The development database integration tests read credentials only from the ignored `infra/env/.env`, and the verification script refuses a target other than `192.168.0.100:5435/swingtrade_db`. If that file or credentials are absent, those database tests skip without attempting a connection. Read tests use the read-only pool. Write-scope tests create a random `go_port_test_<hex>` schema, restrict `search_path` to it, and drop only that schema. They do not run migrations or access existing tables.

## Local commands

- Start the Go skeleton: `./bin/go-run` (default `127.0.0.1:8081`)
- Run unit, real-binary HTTP E2E, optional authorized development-database integration, race, vet, lint, vulnerability and architecture checks: `./bin/go-verify`

The HTTP E2E test builds and starts the actual `swingtrade-api` binary on an ephemeral loopback port and covers every exposed route, including admin authentication and CORS preflight. No container, stage, tunnel, or external service is used by that test.

The verification command enforces at least 80% statement coverage for every tested executable package. The `data/db` package reaches 80.0% through unit tests. With no `infra/env/.env`, the authorized-database integration tests skip, so successful PostgreSQL pool startup and read-only behavior against the live development database remain unverified; the command makes no DB connection. Empty module placeholders have no executable statements or coverage claim.
