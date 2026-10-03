# CI/CD Deployment

## Architecture

### Stage Environment (Docker on pi-node)
Stage runs entirely in Docker containers on pi-node. Monitoring is built in — Prometheus scrapes the API and Grafana provides JVM/Spring Boot dashboards.

| Component | Container | Port (pi-node) | Network |
|-----------|-----------|----------------|---------|
| PostgreSQL | `swing_trade_stage_postgres` | 5436 | swing-trade-stage_swingtrade-network |
| Redis | `swing_trade_stage_redis` | 6380 | swing-trade-stage_swingtrade-network |
| Spring Boot API | `swing-trade-stage-api` | 8081 | swing-trade-stage_swingtrade-network |
| Vue dashboard | `swing-trade-stage-dashboard` | 8082 | swing-trade-stage_swingtrade-network |
| Prometheus | `pi-prometheus` | 9090 | pi-stack_monitoring + swing-trade-stage_swingtrade-network |
| Grafana | `pi-grafana` | 3001 | pi-stack_monitoring |

### Dev Environment (pi-node infra + local app)
Dev runs PostgreSQL on pi-node and the Spring Boot API plus Vue dashboard locally through `dev-stack.sh`. Use the Gradle-built API jar or the stack wrapper; Maven is not the local runtime path. Dev monitoring is available through the API actuator endpoints and local dashboard.

| Component | Location | Port (pi-node) |
|-----------|----------|----------------|
| PostgreSQL | pi-node container | 5435 |
| Redis | pi-node container | 6379 |
| Spring Boot API | Local machine (`dev-stack.sh`) | 8080 |
| Vue dashboard | Local machine (`dev-stack.sh`) | 3003 |

## Stage Deployment

### Stage Deployment (`dev-stack.sh stage`)
```bash
./dev-stack.sh stage
```

Flow:
1. **Build Docker image** — `docker build -f backend/Dockerfile --target runtime-jar ..`
2. **Start infra** — `docker compose -f docker-compose.infra-stage.yml up -d` (PostgreSQL + Redis)
3. **Start API container** — runs on `swing-trade-stage_swingtrade-network` with all stage env vars
4. **Health check** — curls `http://piworm.local:8081/actuator/health`
5. **Prometheus verification** — checks that `swing-trade-stage` target appears in Prometheus

### Deployment path (manual, authoritative)

This project does not have an available GitHub-hosted or self-hosted deployment
runner. The authoritative replacement is the repository-root `dev-stack.sh`
stage command, which builds from the checked-out `stage` worktree, transfers
artifacts to pi-node, starts the stage containers, and verifies API and
Prometheus reachability:

```bash
./dev-stack.sh stage
curl -sf http://piworm.local:8081/actuator/health
curl -sf http://piworm.local:9090/api/v1/targets
# Dashboard smoke check
curl -sfI http://piworm.local:8082/
```

The former self-hosted GitHub Actions definitions are retained for historical
reference under `docs/infra/legacy/` and are not active workflows. Do not use
their queued check status as a deployment signal.

### Container Configuration
```
--name swing-trade-stage-api
--network swing-trade-stage_swingtrade-network
-p 8081:8080
-e SPRING_PROFILES_ACTIVE=stage
-e DB_HOST=swing_trade_stage_postgres
-e DB_NAME=swingtrade_stage
-e REDIS_HOST=swing_trade_stage_redis
```

Key env vars (all configurable via `-e`):
- `DB_HOST`, `DB_PORT`, `DB_NAME` — PostgreSQL connection (resolves to container name on Docker network)
- `REDIS_HOST`, `REDIS_PORT` — Redis connection
- `LLM_BACKEND`, `LLM_PI_SSH_BASE_URL`, `LLAMACPP_MODEL`, `LLM_TIMEOUT` — LLM backend (defaults in `backend/api/src/main/resources/application.properties`)
- `TRADING_ENABLED`, `PAPER_TRADING_ENABLED`, `REAL_TRADING_ENABLED` — trading mode
- `STRATEGY_ENABLED` — signal generation
- `SIGNAL_ENABLED`, `DISCORD_WEBHOOK_ENABLED` — notification toggles

The stage API has a 1 GiB Docker memory cap and a 512 MiB JVM heap maximum
(`-Xmx512m`). The live container was observed without a Docker cap despite the
Compose declaration, so `dev-stack.sh stage` applies the same limit with
`docker update` and verifies `HostConfig.Memory` after startup. The heap maximum
uses half of the container budget, leaving about 512 MiB for metaspace, thread
stacks, direct buffers, code cache, and other native memory. This is a
conservative starting budget: the prior process reached about 1.42 GiB RSS and
observed heap usage peaked near 1 GB, so a 1 GiB cap may still be tight and the
smaller heap may cause more garbage collection or fail under peak load. The
measurements were observational, not a controlled peak-load sizing test.

After deploying this change, check Docker `HostConfig.Memory` is 1073741824,
JVM `Runtime.maxMemory` (or `jvm_memory_max_bytes`) is about 512 MiB, and cold
startup duration completes within the 75-second poll window. Also re-measure
process RSS/high-water, JVM heap used, garbage collection, and OOM/restart
signals under representative stage load. Confirm the cap leaves safe operating
headroom before changing these bounds.

Validation for this change is static only: the Compose values and shell syntax
are checked, but stage is not deployed or cold-started during validation.

### Docker Compose Files
- **`docker-compose.infra-stage.yml`** — PostgreSQL + Redis + API service definition
- **`docker-compose.monitoring-stage.yml`** — Local Grafana (optional, bind-mount issue on Docker Desktop Mac)
- **`backend/Dockerfile`** — Multi-stage: Maven build → `runtime-jar` (JRE 21 Alpine)

## Monitoring Stack

### Prometheus (pi-node)
- **Config**: `/home/dietpi/pi-stack/prometheus/prometheus.yml` (mounted into `pi-prometheus`)
- **Scrape targets**:
  - `prometheus` — self-monitoring
  - `node` — node_exporter (system metrics)
  - `grafana` — Grafana health
  - `caddy` — Caddy reverse proxy metrics
  - `swing-trade-stage` — Spring Boot Actuator `/actuator/prometheus`
  - `vllm` — GPU cluster metrics (external, HTTPS)

### Grafana (pi-node)
- **URL**: http://piworm.local:3001
- **Default creds**: admin / (see `/home/dietpi/pi-stack/grafana/admin-password`)
- **Setup**: Add Prometheus datasource at `http://localhost:9090`, import `jvm-spring-boot.json` dashboard
- **Dashboard**: `backend/monitoring/grafana/dashboards/jvm-spring-boot.json` — 9 panels:
  1. JVM Memory Usage (gauge)
  2. HTTP Request Latency P95 (timeseries)
  3. HTTP Request Rate (timeseries)
  4. GC Pause Time (timeseries)
  5. HikariCP Connection Pool (timeseries)
  6. JVM Threads (timeseries)
  7. HTTP 4xx/5xx Error Rates (timeseries)
  8. Heap Memory Usage (timeseries)
  9. System CPU Usage (timeseries)

### Local Grafana (optional)
`./dev-stack.sh stage-monitoring up -d` — runs Grafana on port 3002, configured to scrape pi-node Prometheus at `http://piworm.local:9090`. Bind mounts don't work reliably on Docker Desktop Mac, so provisioning files must be baked into the image or configured manually.

## Infrastructure on pi-node

### Volumes
| Volume | Database |
|--------|----------|
| `postgres_data` | dev (swingtrade_db) |
| `postgres_data_stage` | stage (swingtrade_stage) |
| `redis_data` | dev |
| `redis_data_stage` | stage |

### Networks
| Network | Purpose |
|---------|---------|
| `swingtrade-network` | dev services |
| `swing-trade-stage_swingtrade-network` | stage services + API + Prometheus |
| `swing-trade-dev_swingtrade-network` | dev services |
| `pi-stack_monitoring` | Prometheus + Grafana + node_exporter + caddy |

### Docker Contexts
| Context | Target | Use |
|---------|--------|-----|
| `desktop-linux` | localhost Docker Desktop | Local development |
| `pi-node` | ssh://dietpi@piworm.local | Remote infra + stage |

Switch context: `docker context use pi-node` / `docker context use desktop-linux`

## Known Issues

1. **Fyers SDK** — force-added to git (ignored by *.jar), installed by `dev-stack.sh stage`
2. **Maven cache corruption** — historical workflow issue; the active stage path uses Gradle
3. **Log directory permissions** — changed from `/var/log/swing-trade/` (needs sudo) to `/home/dietpi/swing-trade/logs/`
4. **Type mismatch** — `findUnprocessedBuySignalsSince` took `LocalDateTime` but `SignalEntity.date` is `LocalDate` (fixed in SignalRepository.java:168)
5. **Docker bind mounts on Mac** — Docker Desktop Mac can't access bind-mounted paths from its VM. Local Grafana provisioning volumes fail. Use pi-grafana instead.
6. **Container names with underscores** — Tomcat rejects Host headers with underscores. Use hyphenated names (e.g., `swing-trade-stage-api`).
7. **GitHub Actions runners** — historical self-hosted definitions are archived; stage deployment is performed and verified through `dev-stack.sh stage`.
