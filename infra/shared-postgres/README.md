# Shared Postgres stack (pg-dev / pg-prod)

Self-contained compose for the shared Postgres containers approved in the consolidation design. One database plus least-privilege roles per project per environment.

## Files

- `compose.dev.yml` - `pg-dev` on `192.168.0.100:5433`
- `compose.prod.yml` - `pg-prod` on `192.168.0.100:5434`
- `config/postgresql.conf`, `config/pg_hba.conf` - mounted into both containers
- `bootstrap/bootstrap.sh` - idempotent role/database bootstrap
- `bootstrap/test-isolation.sh` - proves each app role reaches only its own database
- `bootstrap/restore-check.sh` - verifies a backup dump restores into a throwaway database

## Secrets (never in repository)

Generated on the Pi in mode-600 env files outside any repo:

- `/home/dietpi/shared-postgres/pg-dev.env`
- `/home/dietpi/shared-postgres/pg-prod.env`

Key names:

Dev: `PG_DEV_SUPERUSER_PASSWORD`, `PG_DEV_DATABASES`, `SWINGTRADE_APP_PASSWORD`, `SWINGTRADE_MIGRATION_PASSWORD`, `FINAPP_APP_PASSWORD`, `FINAPP_MIGRATION_PASSWORD`, `IRONVEIL_DEV_APP_PASSWORD`, `IRONVEIL_DEV_MIGRATION_PASSWORD`, `GRAFANA_RO_PASSWORD`

Prod: `PG_PROD_SUPERUSER_PASSWORD`, `PG_PROD_DATABASES`, `SWINGTRADE_APP_PASSWORD`, `SWINGTRADE_MIGRATION_PASSWORD`, `FINAPP_APP_PASSWORD`, `FINAPP_MIGRATION_PASSWORD`, `IRONVEIL_APP_PASSWORD`, `IRONVEIL_MIGRATION_PASSWORD`, `GRAFANA_RO_PASSWORD`

Values are random hex strings. They are never printed, committed, or logged.

## TLS

A small internal CA and server certificate are generated on the Pi under `/home/dietpi/shared-postgres/tls/` (`ca.crt`, `ca.key`, `server.crt`, `server.key`) and mounted read-only into both containers. `ca.key` and `server.key` are mode 600.

Clients get the CA certificate with:

```
scp dietpi@piworm.local:/home/dietpi/shared-postgres/tls/ca.crt ./pg-shared-ca.crt
```

Then connect with `sslmode=verify-full` and `sslrootcert=./pg-shared-ca.crt`. The server certificate covers `192.168.0.100`, `pg-dev`, `pg-prod`, and `localhost`.

## Operation

Run via `docker --context pi-node compose --file compose.dev.yml --env-file /home/dietpi/shared-postgres/pg-dev.env up -d` (env file lives on the Pi; fetch through your normal SSH flow). From the Pi host itself the same commands work with local `docker`.
