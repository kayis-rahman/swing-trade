#!/usr/bin/env bash
set -u

ENVIRONMENT=${1:?usage: test-isolation.sh <dev|prod> <env-file>}
ENV_FILE=${2:?}
DOCKER=${DOCKER:-docker}

set -a; . "$ENV_FILE"; set +a

allowed() {
  # returns 0 if role may connect to db
  local role="$1" db="$2"
  case "$ENVIRONMENT:$role:$db" in
    dev:swingtrade_app:swingtrade|dev:swingtrade_app:swingtrade_stage) return 0 ;;
    dev:finapp_app:fin_app) return 0 ;;
    dev:ironveil_app:ironveil_dev|dev:ironveil_app:ironveil_test) return 0 ;;
    prod:swingtrade_app:swingtrade) return 0 ;;
    prod:finapp_app:fin_app) return 0 ;;
    prod:ironveil_app:ironveil) return 0 ;;
    *) return 1 ;;
  esac
}
pw_for() {
  local role="$1"
  case "$role" in
    swingtrade_app) echo "$SWINGTRADE_APP_PASSWORD" ;;
    finapp_app) echo "$FINAPP_APP_PASSWORD" ;;
    ironveil_app)
      if [[ "$ENVIRONMENT" == "dev" ]]; then echo "$IRONVEIL_DEV_APP_PASSWORD"; else echo "$IRONVEIL_APP_PASSWORD"; fi
      ;;
  esac
}

case "$ENVIRONMENT" in
  dev) CONTAINER=pg-dev; DBS="swingtrade swingtrade_stage fin_app ironveil_dev ironveil_test" ;;
  prod) CONTAINER=pg-prod; DBS="swingtrade fin_app ironveil" ;;
  *) echo "environment must be dev or prod" >&2; exit 2 ;;
esac

failed=0
for role in swingtrade_app finapp_app ironveil_app; do
  pw="$(pw_for "$role")"
  for db in $DBS; do
    if $DOCKER exec -e "PGPASSWORD=${pw}" "$CONTAINER" psql -U "$role" -d "$db" -c "SELECT 1" >/dev/null 2>&1; then
      got=1
    else
      got=0
    fi
    if allowed "$role" "$db"; then want=1; else want=0; fi
    if [[ "$want" == "$got" ]]; then
      if [[ "$want" == "1" ]]; then echo "PASS: ${role} can connect to ${db}"; else echo "PASS: ${role} denied on ${db}"; fi
    else
      echo "FAIL: ${role} allowed=${want} connected=${got} db=${db}"
      failed=1
    fi
  done
done
exit "$failed"
