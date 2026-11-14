# Scratch Postgres Go parity runbook

## Purpose

Phase 6 step 4: run the Go API against a disposable Postgres 16, exercise the
available read and persisted-state surfaces, and record differences against Java.

## Database (disposable)

1. Initialise a fresh cluster in a temp data directory.
2. Start it on 127.0.0.1:55433 with `SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1`.
3. Create role and database `go_port_test` (no password).
4. Apply every Flyway migration under `backend/data/src/main/resources/db/migration`.
5. Grant `go_port_test` privileges.

Do not use Docker, shared pg-dev, stage or prod.

## Go start

```bash
export SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1
export SWINGTRADE_LOCAL_TEST_DATABASE_URL="postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
DATABASE_URL="not-a-postgres-url" \
SERVER_PORT=8090 go run ./go/api/cmd/swingtrade-api
```

## Java

JDK 21 is available as `/opt/homebrew/opt/openjdk@21`. Run Java and Go side by side against the
same cluster and compare them with `scripts/java-go-parity.py`; the commands and the
2026-11-14 results are in [java-go-endpoint-parity.md](java-go-endpoint-parity.md).

## Cleanup

Stop `pg_ctl`, delete the data directory.
