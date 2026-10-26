# Watchlist shadow comparison runbook

Captain-authorized shadow comparison for the two read-only watchlist endpoints:
nginx forwards the same `GET /api/watchlist` and `GET /api/watchlist/all`
requests to Java and to a Go shadow service, Java's response stays primary, the
mirrored Go response is discarded, and the comparator reports differences. The
one-trading-week comparison run is a release-readiness gate; this runbook is its
documented setup and is not a record that the run happened.

## Components

| Piece | Location | Default state |
|---|---|---|
| Mirror wiring (resolver + maps + two exact locations) | `dashboard/nginx.conf` (image built by `dashboard/Dockerfile`) and `infra/nginx/dashboard-nginx/default.conf` (copied by `dev-stack.sh stage`) | Disabled: both map entries are comments |
| Go shadow service | `infra/docker-compose.go-shadow.yml`, container `swing-trade-go-shadow-api` on `swing-trade_swingtrade-network` (the running dashboard's network) | Not running |
| Comparator | `go/api/parity`, env-driven test `TestConfiguredReadParity` | Skips unless both base URLs are set |

The stage deployment compose file `infra/docker-compose.go-stage.yml` is
untouched by this setup: `bin/verify-go-stage-deployment.sh` keeps asserting its
container name and `8081:8080` port mapping.

## Disabled by default

Shipped, with mirroring off - the second map ends in `default "";` and both
entries are comments:

```nginx
map "$request_method $watchlist_mirror_path" $watchlist_mirror_target {
    default "";
    # "GET /api/watchlist"     http://swing-trade-go-shadow-api:8080;
    # "GET /api/watchlist/all" http://swing-trade-go-shadow-api:8080;
}
```

While disabled, a request to either watchlist path still reaches Java, and the
internal `= /go_watchlist_mirror` location answers `204` without contacting any
backend - a default deploy never sends a byte to the Go shadow service.

## Activation

Activation is exactly one edit, performed only when the shadow service is
running and the comparison is authorized: in **both** active nginx config files
(`dashboard/nginx.conf` and `infra/nginx/dashboard-nginx/default.conf`), delete
the leading `# ` from these two lines inside the `watchlist_mirror_target` map:

```nginx
    "GET /api/watchlist"     http://swing-trade-go-shadow-api:8080;
    "GET /api/watchlist/all" http://swing-trade-go-shadow-api:8080;
```

Rebuild and restart the dashboard image so the edited config is baked in
(`dev-stack.sh stage` for the stage flow, or a plain
`docker build` of `dashboard/`). Only those two GET paths are mirrored: the map
keys on method plus request path, so `POST /api/watchlist` and every other route
still only reach Java. `resolver 127.0.0.11 valid=5s` supplies runtime DNS for
the variable `proxy_pass` (Docker's embedded resolver); the shadow container
name is the network alias the map targets.

To deactivate, re-comment the two lines and rebuild the dashboard image. No
state persists from mirroring.

## Starting the shadow service

The shadow image packages a prebuilt binary exactly like the gated stage
deploy, from a scratch directory (never from the worktree, so no binary lands in
the repository). On the Docker host:

```bash
mkdir -p /tmp/go-shadow
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" -o /tmp/go-shadow/swingtrade-api ./go/api/cmd/swingtrade-api
cp infra/Dockerfile.go-stage infra/docker-compose.go-shadow.yml /tmp/go-shadow/
cd /tmp/go-shadow
docker compose -f docker-compose.go-shadow.yml build
API_KEY=<stage api key> \
DATABASE_URL='postgresql://<user>:<pass>@192.168.0.100:5435/swingtrade_db?sslmode=disable' \
  docker compose -f docker-compose.go-shadow.yml up -d
```

(`arm64` is the Pi; use `amd64` on an x86 Docker host. Run these from a
repository checkout; for a remote Docker host, cross-compile on the workstation
and copy the binary, `infra/Dockerfile.go-stage`, and the shadow compose file
into the scratch directory there, the way `bin/deploy-go-stage.sh` does for
stage.)

Boundary: the Go pool guard only accepts the authorized development database
(`192.168.0.100:5435/swingtrade_db`), so a meaningful comparison requires the
Java side to read that same database. A stage-soak against the stage database
additionally needs a stage-database authorization for Go and a stage deployment
authorization; this runbook grants neither, and `bin/deploy-go-stage.sh` keeps
refusing stage replacement. No stage or production deployment is part of this
document.

## Running the comparator

The comparator sends the same read requests to both services and prints every
difference (status, content type, body; `/api/watchlist/all` records are matched
order-insensitively by id because neither contract fixes array order):

```bash
JAVA_SERVICE_URL=http://127.0.0.1:8080 \
GO_SERVICE_URL=http://127.0.0.1:8083 \
  go test ./go/api/parity -run TestConfiguredReadParity -v
```

`JAVA_SERVICE_URL` may point at any Java instance reading the same database as
the Go target (local dev Java on `8080`, or a same-DB stage Java), and
`GO_SERVICE_URL` may be a locally started Go service instead of the shadow
container (`./bin/go-run` serves on `127.0.0.1:8081`). Run it once before
enabling the map so the two sides are known-good, then leave it running or
re-run it daily during the mirror period.

## The one-trading-week run

With the map activated and the shadow service up, leave the dashboard serving
normal traffic for one full trading week, collecting comparator output. The
week-long soak is a release-readiness gate, not a merge gate: deterministic
comparator runs do not substitute for it, and this runbook does not claim the
week has been run. When the week completes, record the report and either keep
mirroring or deactivate as above.
