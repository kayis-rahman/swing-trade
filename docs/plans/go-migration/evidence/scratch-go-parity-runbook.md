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
```
export SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1
export SWINGTRADE_LOCAL_TEST_DATABASE_URL="postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
DATABASE_URL="not-a-postgres-url" \
SERVER_PORT=8090 ./go/api/cmd/swingtrade-api (or the built binary)
```

## Java
A Java 21 JDK is not installed in this environment, so the comparator could not
start the backend. Java recorded outputs in `docs/plans/go-migration/evidence/goldens`
remain the comparison baseline; re-run against a Java instance once JDK is available.

## Cleanup
Stop `pg_ctl`, delete the data directory.
