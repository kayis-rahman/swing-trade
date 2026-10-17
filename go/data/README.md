# data

Go module `github.com/kayis-rahman/swing-trade/go/data`, corresponding to Gradle module `backend/data`. Owns PostgreSQL access, repositories, stores, market-data clients and ingestion services. May import `github.com/kayis-rahman/swing-trade/go/core` only.

Package layout mirrors the Java `entity`, `repository`, `store`, `client` and `service` roles. Database and provider calls do not enter `core` or `strategy`.

The initial pgx pool forces `default_transaction_read_only=on`. Integration tests are opt-in through the authorized development credentials in ignored `infra/env/.env`; any write test uses a generated `go_port_test_<random>` schema, restricts its `search_path`, and drops only that schema. No migrations or writes to existing/public data are run.
