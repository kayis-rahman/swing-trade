# data

Go module `github.com/kayis-rahman/swing-trade/go/data`, corresponding to Gradle module `backend/data`. Owns PostgreSQL access, repositories, stores, market-data clients and ingestion services. May import `github.com/kayis-rahman/swing-trade/go/core` only.

Package layout mirrors the Java `entity`, `repository`, `store`, `client` and `service` roles. Database and provider calls do not enter `core` or `strategy`.

The initial pgx pool forces `default_transaction_read_only=on`. Watchlist, NSE holiday, settings, and latest-signal reads use sqlc-generated queries; regenerate them with `cd go/data && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate -f sqlc.yaml`. `sqlc/schema.sql` is the schema subset for that package, kept aligned with the authoritative Flyway migration. Default integration tests read only the authorized Spring datasource settings from ignored `infra/env/.env` and reject targets outside the approved development database. Write tests create a generated `go_port_test_<random>` schema, restrict `search_path` to it, and drop only that schema. When the host process cannot route to the dev DB, `SWINGTRADE_USE_LOCAL_TEST_DATABASE=1 ./bin/go-verify` opts into a fixed loopback-only test target at `127.0.0.1:55433`; this test-only path does not relax the production pool guard. No migrations or public-schema writes are run.
