# Local Go stack with frontend

The dashboard talks to the Go service on `127.0.0.1:8081`; Java is not started. `dashboard/vite.config.ts` proxies `/api` and `/fyers` to `http://localhost:8081`, so every browser request to `http://localhost:3003/api/...` lands on Go.

## Start

Build the database URL from the authorized development datasource settings in the gitignored `infra/env/.env`. Read only `SPRING_DATASOURCE_URL`, `SPRING_DATASOURCE_USERNAME`, and `SPRING_DATASOURCE_PASSWORD`; never log the derived URL.

```bash
cd /Users/kayisrahman/.treehouse/swing-trade-3a2f1e/1/swing-trade

# Derive DATABASE_URL in this shell only (authorized target: 192.168.0.100:5435/swingtrade_db).
export DATABASE_URL="$(python3 - <<'PY'
import pathlib, urllib.parse
settings = {}
for line in pathlib.Path("infra/env/.env").read_text().splitlines():
    if "=" not in line or line.lstrip().startswith("#"):
        continue
    key, value = line.split("=", 1)
    settings[key.strip()] = value.strip().strip('"').strip("'")
url = urllib.parse.urlsplit(settings["SPRING_DATASOURCE_URL"].removeprefix("jdbc:"))
user = urllib.parse.quote(settings["SPRING_DATASOURCE_USERNAME"], safe="")
password = urllib.parse.quote(settings["SPRING_DATASOURCE_PASSWORD"], safe="")
print(f"postgresql://{user}:{password}@{url.hostname}:{url.port}{url.path}?sslmode=disable")
PY
)"

./bin/go-run
```

`./bin/go-run` serves on `127.0.0.1:8081` and exposes `javaVersion: "go<version>"` in `/api/health`, which is how the runbook proves the backend is Go.

In a second terminal:

```bash
cd /Users/kayisrahman/.treehouse/swing-trade-3a2f1e/1/swing-trade/dashboard
yarn dev
```

The dashboard is at `http://localhost:3003/`.

## Evidence

Captured 2026-10-28 on `fm/swing-trade-go-migration-impl-v2` with Java stopped:

```bash
curl -s http://127.0.0.1:8081/api/health | head -c 200
# {"components":{"api":{...}},"status":"UP","systemInfo":{"javaVersion":"go1.27.3",...}}

curl -s http://localhost:3003/api/health | head -c 200
# same body: the Vite proxy is forwarding to Go, not Java

curl -s http://localhost:3003/api/signals/latest | python3 -c 'import json,sys; d=json.load(sys.stdin); print(type(d).__name__, len(d))'
# list 59   (bare JSON array, Java SignalController shape)

curl -s http://localhost:3003/api/watchlist | head -c 120
# {"data":[{"addedAt":"...","candleCount":738,...}]
```

Browser check: `http://localhost:3003/signals` renders 59 signals (matching the API count) with the left-nav "Signals" badge and the footer "System Status: Healthy". A screenshot is kept at `docs/plans/go-migration/evidence/frontend-go-stack-2026-10-28.png`.
