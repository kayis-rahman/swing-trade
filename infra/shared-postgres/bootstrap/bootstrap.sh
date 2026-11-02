#!/usr/bin/env bash
set -euo pipefail

ENVIRONMENT=${1:?usage: bootstrap.sh <dev|prod> <env-file>}
ENV_FILE=${2:?usage: bootstrap.sh <dev|prod> <env-file>}
DOCKER=${DOCKER:-docker}

if [[ "$ENVIRONMENT" != "dev" && "$ENVIRONMENT" != "prod" ]]; then
  echo "environment must be dev or prod" >&2
  exit 2
fi
if [[ ! -f "$ENV_FILE" ]]; then
  echo "env file not found: $ENV_FILE" >&2
  exit 2
fi

# Source only variable assignments; never print them.
set -a
. "$ENV_FILE"
set +a

case "$ENVIRONMENT" in
  dev)
    CONTAINER=pg-dev
    DATABASES="swingtrade swingtrade_stage fin_app ironveil_dev ironveil_test"
    PG_SUPER_PASSWORD_VAR=PG_DEV_SUPERUSER_PASSWORD
    ;;
  prod)
    CONTAINER=pg-prod
    DATABASES="swingtrade fin_app ironveil"
    PG_SUPER_PASSWORD_VAR=PG_PROD_SUPERUSER_PASSWORD
    ;;
esac

PG_SUPER_PASSWORD="${!PG_SUPER_PASSWORD_VAR:?missing $PG_SUPER_PASSWORD_VAR}"
PGPORT_BIND=$([[ "$ENVIRONMENT" == "dev" ]] && echo 5433 || echo 5434)

psql_super() {
  # Reads SQL from stdin. Password is passed via PGPASSWORD (not logged).
  $DOCKER exec -i -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" \
    psql -U postgres -d postgres -v ON_ERROR_STOP=1 "$@"
}

echo "Bootstrapping $ENVIRONMENT databases: $DATABASES"

# Ensure each database exists (CREATE DATABASE cannot run in a transaction, so guard in shell).
for db in $DATABASES; do
  exists="$( $DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" \
    psql -U postgres -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${db}'" )"
  if [[ "$exists" != "1" ]]; then
    $DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" \
      createdb -U postgres "$db"
    echo "created database $db"
  fi
done

# Create shared login roles idempotently per environment. Project keys are fixed.
# Expected env keys: SWINGTRADE_APP_PASSWORD, SWINGTRADE_MIGRATION_PASSWORD,
# FINAPP_APP_PASSWORD, FINAPP_MIGRATION_PASSWORD,
# IRONVEIL_DEV_APP_PASSWORD, IRONVEIL_DEV_MIGRATION_PASSWORD (dev) etc.
# Ironveil prod uses IRONVEIL_APP_PASSWORD/IRONVEIL_MIGRATION_PASSWORD.
# Grafana: GRAFANA_RO_PASSWORD.
GRAFANA_PASSWORD="${GRAFANA_RO_PASSWORD:?missing GRAFANA_RO_PASSWORD}"

# Role prefix helper: role names are <project>_app / <project>_migration / <project>_ro.
create_role() {
  local rolename="$1"
  local password="$2"
  local attrs="$3" # extra attributes
  exists="$($DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='${rolename}'" 2>/dev/null || true)"
  if [[ "${exists}" != "1" ]]; then
    sql="CREATE ROLE ${rolename} LOGIN PASSWORD '${password}' ${attrs} NOCREATEROLE NOSUPERUSER;"
  else
    sql="ALTER ROLE ${rolename} PASSWORD '${password}';"
  fi
  $DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "${sql}"
}

# Determine Ironveil credential key prefix.
if [[ "$ENVIRONMENT" == "dev" ]]; then
  IRONVEIL_APP="${IRONVEIL_DEV_APP_PASSWORD:?missing}"
  IRONVEIL_MIG="${IRONVEIL_DEV_MIGRATION_PASSWORD:?missing}"
else
  IRONVEIL_APP="${IRONVEIL_APP_PASSWORD:?missing}"
  IRONVEIL_MIG="${IRONVEIL_MIGRATION_PASSWORD:?missing}"
fi

create_role swingtrade_app "${SWINGTRADE_APP_PASSWORD:?missing}" ""
create_role swingtrade_migration "${SWINGTRADE_MIGRATION_PASSWORD:?missing}" "CREATEDB"
create_role finapp_app "${FINAPP_APP_PASSWORD:?missing}" ""
create_role finapp_migration "${FINAPP_MIGRATION_PASSWORD:?missing}" "CREATEDB"
create_role ironveil_app "${IRONVEIL_APP:?missing}" ""
create_role ironveil_migration "${IRONVEIL_MIG:?missing}" "CREATEDB"
create_role grafana_ro "${GRAFANA_PASSWORD}" ""

# Revoke PUBLIC connect on every database (must be managed from postgres DB).
for db in $DATABASES; do
  $DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "REVOKE ALL ON DATABASE ${db} FROM PUBLIC; REVOKE CONNECT ON DATABASE ${db} FROM PUBLIC;"
done

# Make migration role database owner first so subsequent ACL changes persist.
for db in $DATABASES; do
  case "$db" in
    swingtrade|swingtrade_stage) owner=swingtrade_migration ;;
    fin_app) owner=finapp_migration ;;
    ironveil_dev|ironveil_test|ironveil) owner=ironveil_migration ;;
  esac
  $DOCKER exec -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d postgres -v ON_ERROR_STOP=1 \
    -c "ALTER DATABASE ${db} OWNER TO ${owner};"
done

# Per-database grants: app DML, migration owner-ish, grafana read.
grant_db() {
  local db="$1"
  local app="$2"
  local migration="$3"
  local ironveil=0
  # Database-level ACL must be changed while connected to a different database.
  $DOCKER exec -i -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d postgres -v ON_ERROR_STOP=1 <<SQL
GRANT CONNECT ON DATABASE ${db} TO ${app}, ${migration}, grafana_ro;
REVOKE ALL ON DATABASE ${db} FROM PUBLIC;
REVOKE CONNECT ON DATABASE ${db} FROM PUBLIC;
SQL
  $DOCKER exec -i -e "PGPASSWORD=${PG_SUPER_PASSWORD}" "$CONTAINER" psql -U postgres -d "$db" -v ON_ERROR_STOP=1 <<SQL
GRANT USAGE ON SCHEMA public TO ${app}, ${migration}, grafana_ro;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ${app};
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ${app};
GRANT ALL PRIVILEGES ON SCHEMA public TO ${migration};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ${migration};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ${migration};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO grafana_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO grafana_ro;
SQL
  # Make app/migration roles owners of future? migration created objects owned by migration due CREATEDB? Owner defaults to creating role at runtime anyway.
}

for db in $DATABASES; do
  case "$db" in
    swingtrade|swingtrade_stage) grant_db "$db" swingtrade_app swingtrade_migration ;;
    fin_app) grant_db "$db" finapp_app finapp_migration ;;
    ironveil_dev|ironveil_test|ironveil) grant_db "$db" ironveil_app ironveil_migration ;;
  esac
done

echo "Bootstrap complete for $ENVIRONMENT"
