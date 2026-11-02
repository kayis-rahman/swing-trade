#!/usr/bin/env bash
set -euo pipefail

ENVIRONMENT=${1:?usage: restore-check.sh <dev|prod> <env-file> <dbname> <dump-filename>}
ENV_FILE=${2:?}
DB_NAME=${3:?}
DUMP_FILE=${4:?}
DOCKER=${DOCKER:-docker}

set -a; # shellcheck source=/dev/null
. "$ENV_FILE"; set +a

case "$ENVIRONMENT" in
  dev) CONTAINER=pg-dev-backup ;;
  prod) CONTAINER=pg-prod-backup ;;
  *) echo "environment must be dev or prod" >&2; exit 2 ;;
esac

SCRATCH_DB="restore_check_$(date -u +%Y%m%d%H%M%S)"
cleanup() {
  $DOCKER exec "$CONTAINER" dropdb --if-exists "$SCRATCH_DB" >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

# Validate dump file exists and is a valid custom-format archive.
$DOCKER exec "$CONTAINER" sh -c 'test -s "/backups/'"$DUMP_FILE"'" && pg_restore --list "/backups/'"$DUMP_FILE"'" >/dev/null'
echo "dump validated: $DUMP_FILE"

$DOCKER exec "$CONTAINER" createdb "$SCRATCH_DB"
START=$(date +%s)
$DOCKER exec "$CONTAINER" pg_restore --exit-on-error --no-owner --no-privileges -d "$SCRATCH_DB" "/backups/${DUMP_FILE}"
END=$(date +%s)
ELAPSED=$((END-START))
COUNT=$($DOCKER exec "$CONTAINER" psql -d "$SCRATCH_DB" -Atqc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")
echo "restore check passed: db=${DB_NAME} dump=${DUMP_FILE} tables=${COUNT} elapsed=${ELAPSED}s (scratch db dropped)"
