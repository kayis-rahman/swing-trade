# Log fields and consumers inventory (Phase 0, step 9)

Status: complete, including the runtime half. The static half was derived from the single logback
configuration, the three property files, `TraceIdFilter`, the deploy scripts and the Spring Boot
4.1.1 logback integration; the runtime half was confirmed live on pi-node (promtail config, Loki
streams, the on-disk stage log file) on 2026-10-16. No product code was changed.

Derived by: reading `backend/api/src/main/resources/logback-spring.xml` (the only logback or
log4j configuration in the repo), `application.properties:242-259`,
`application-local.properties:124-143`, `application-stage.properties:94-114`,
`backend/api/src/main/java/com/swingtrade/api/config/TraceIdFilter.java` (the only MDC writer in
main code), `dev-stack.sh` (start/stop/logs-json/live-logs), `infra/docker-compose.infra-stage.yml`
(env vars, volume), `infra/legacy/github-actions-deploy-main-self-hosted.yml`, the Spring Boot
4.1.1 `spring-boot.jar` logback resources (`defaults.xml`, `base.xml`, `file-appender.xml`,
`RollingPolicySystemProperty`), and `infra/monitoring/**` plus the node-local monitoring configs
read over SSH (read-only).

## 1. Summary

| Measure | Value |
|---|---|
| Logback configurations in the repo | 1 (`backend/api/src/main/resources/logback-spring.xml`); no log4j2, no `logging.structured.*` anywhere |
| Appenders | 2 on the root logger for both `local` and `stage` profiles: `CONSOLE` and `ASYNC_FILE` (wraps `FILE`); every event is formatted twice and written twice |
| MDC fields | 3: `traceId`, `symbol`, `module` - all written only by `TraceIdFilter` on HTTP request threads; empty on every scheduler/background thread |
| File sinks | local: `backend/logs/swing-trade-local.log` (via `LOG_FILE` env from `dev-stack.sh`); stage: `/var/log/swing-trade/stage.log` in the `stage_logs` named volume |
| Rolling | 50 MB per file, daily + size index, 7 files kept, 2 GB total cap, **uncompressed** (pattern does not end in `.gz`); the stage property `logging.file.max-size=100MB` / `max-history=30` are **dead** (see §2) |
| Consumers | promtail (docker stdout only, via `docker_sd`; node-local config, not in repo) -> Loki (168 h retention) -> Grafana (datasource provisioned, **zero panels** query it); `dev-stack.sh logs-json` (expects JSON lines, currently always falls through to a raw `tail`); humans via `docker logs` / `docker exec tail`; legacy GH-Actions `cat` of a host-run jar log (archived) |
| Consumers of the MDC fields | **none** - the file pattern is the only sink that prints them, and they are always empty in the production file |

## 2. Configuration surface and what is actually in force

`logback-spring.xml` includes `org/springframework/boot/logging/logback/defaults.xml` (color
converters, `CONSOLE_LOG_PATTERN`/`FILE_LOG_PATTERN` variable defaults, and the quiet-logger
table in §5), then defines its own `CONSOLE` appender (`${CONSOLE_LOG_PATTERN:-<plain fallback>}`)
and its own `FILE` `RollingFileAppender` with a **hard-coded** pattern and rolling policy.

Spring Boot 4.1.1 binding rules that matter here (verified from `spring-boot-4.1.1.jar`):

- Because a `logback-spring.xml` exists, Spring Boot loads only that file. `base.xml` (which
  carries Spring Boot's own `FILE` appender using `${FILE_LOG_PATTERN}`,
  `${LOGBACK_ROLLINGPOLICY_*}`) is **not** loaded, so:
  - `logging.pattern.file=...` in `application-stage.properties:114` is **dead** - it sets the
    `FILE_LOG_PATTERN` variable, which only the never-loaded `base.xml`/`file-appender.xml`
    reference. The file pattern is the hard-coded one in `logback-spring.xml:16`.
  - `logging.file.max-size=100MB` and `logging.file.max-history=30`
    (`application-stage.properties:111-112`) map to the `LOGBACK_ROLLINGPOLICY_MAX_FILE_SIZE` /
    `LOGBACK_ROLLINGPOLICY_MAX_HISTORY` system properties (enum
    `RollingPolicySystemProperty`), which likewise only the never-loaded `file-appender.xml`
    references. **Dead.** The effective policy is the hard-coded 50 MB / 7 / 2 GB.
  - `logging.pattern.console` is **live**: it sets `CONSOLE_LOG_PATTERN`, which the app's own
    `CONSOLE` appender references.
  - `logging.file.name` sets the `LOG_FILE` logback variable, which the app's `FILE` appender
    references (with its own `:-` fallback).
- `logging.level.*` entries are applied by Spring after the config file loads, so where a
  `springProfile` block in `logback-spring.xml` and a `logging.level.*` property set the same
  logger, the property wins. In practice both mechanisms agree (see §5).

Environment-variable chain for the file sink:

| Environment | `LOG_FILE` | Resulting file | Set by |
|---|---|---|---|
| local dev (`dev-stack.sh start`) | `$BACKEND_DIR/logs/swing-trade-local.log` | `backend/logs/swing-trade-local.log` (CWD is `backend/`) | `dev-stack.sh:110` |
| stage container | `/var/log/swing-trade/stage.log` | same, inside the `stage_logs` named volume | `infra/docker-compose.infra-stage.yml:47` |
| fallback (no env) | - | `./logs/swing-trade-local.log` | `logback-spring.xml:12,14` and `application.properties:259` |

Both local dev (`dev-stack.sh:114`, `-Duser.timezone=Asia/Kolkata`) and the stage container
(`infra/docker-compose.infra-stage.yml:49`, `TZ=Asia/Kolkata`) pin the JVM to IST, so log
timestamps are IST in every real deployment.

## 3. Line formats, per sink and environment

| Sink | local profile | stage profile | other profiles |
|---|---|---|---|
| console | `%d{yyyy-MM-dd HH:mm:ss} %-5level %logger{36} - %msg%n` (`application-local.properties:139`, no thread name) | `%d{yyyy-MM-dd HH:mm:ss} [%thread] %-5level %logger{36} - %msg%n` (`application-stage.properties:113`) | Spring Boot 4 default with ANSI colors: `%clr(%d{ISO8601+zone}){faint} %clr(%5p) ... --- [name] [thread] logger : %m%n%wEx` |
| file | identical pattern for both profiles, hard-coded in `logback-spring.xml:16` (see below) | same | n/a - the file appender exists under `local` and `stage` profile blocks only |

The hard-coded FILE pattern is:

```text
%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n traceId=%X{traceId} symbol=%X{symbol} module=%X{module}
```

Two defects make the physical file layout non-obvious (both confirmed on the live stage file):

1. `%n` sits **after** `%msg` and **before** the MDC stub, so the stub starts a new physical line.
2. There is **no trailing newline** after `module=`, so the stub of event K is concatenated with
   the full content of event K+1 on the same physical line.

The on-disk result (verbatim from `tail -8 .../stage.log | cat -A`, pi-node, 2026-10-16):

```text
 traceId= symbol= module=2026-10-16 18:19:36.166 [job-orchestrator-476] INFO  c.s.llm.service.LlmClientProvider - Selected LLM backend OPENAI with model longcat-2.5-preview-free at https://opencode.ai/zen/go/v1$
 traceId= symbol= module=2026-10-16 18:21:13.445 [job-orchestrator-408] INFO  c.s.a.service.JobOrchestratorService - Run d25f0ff0-a379-4924-9f8c-89da4c66440b completed with status COMPLETED_WITH_WARNINGS$
 traceId= symbol= module=
```

The first physical line of the file is a bare message (no stub prefix), since event 1's stub
merges into event 2. All 495 stub lines in the current stage file carry **empty** values, because
every recent event came from scheduled jobs, whose threads never set MDC (see §4). A line-based
consumer of the file (e.g. `jq`, a naive `grep`) sees the merged stub as leading garbage on every
line.

Console lines are clean single lines: the Loki stream for `swing-trade-stage-api` (queried on the
node) contains exactly the console-pattern lines, one entry per physical line, no MDC content.

## 4. MDC fields

| Field | Value | Set by | Scope |
|---|---|---|---|
| `traceId` | first 8 chars of a `UUID.randomUUID()` | `TraceIdFilter.doFilter`, per request | HTTP request thread only; cleared in `finally` |
| `symbol` | query param `symbol`, else `code`, else the last path segment when the URI contains `/signals/` or `/positions/`; omitted when absent/blank | same | same |
| `module` | constant `"api"` | same | same |

`TraceIdFilter` is a plain `@Component` servlet `Filter` with no URL pattern, so it runs for
**every** request the container receives (including `/actuator/**`), but MDC never exists on
scheduler, async, or job-orchestrator threads. Nothing else in main code calls `MDC` (verified by
grep), and nothing consumes the fields (no log parser, no Loki pipeline stage, no dashboard). The
only sink that prints them is the FILE appender, where they are always empty in production.

`logback-spring.xml` also sets `includeCallerData=true` on the `ASYNC_FILE` appender, which makes
logback capture the caller stack for every event although the pattern uses no `%caller` - pure
overhead. `queueSize=512` with `discardingThreshold=0` means the queue never discards
(`ASYNC_FILE` will block the writer when full).

## 5. Per-profile logger levels

| Logger | base (`application.properties:242-256`) | `application-local.properties:124-136` | `application-stage.properties:94-107` | `logback-spring.xml` springProfile |
|---|---|---|---|---|
| root | INFO | INFO | INFO | INFO (both profiles) |
| `com.swingtrade` | DEBUG | DEBUG | INFO | local: DEBUG, stage: INFO |
| `com.swingtrade.api` | DEBUG | DEBUG | INFO | - |
| `com.swingtrade.data` | DEBUG | DEBUG | INFO | - |
| `com.swingtrade.strategy` | DEBUG | DEBUG | INFO | - |
| `com.swingtrade.llm` | DEBUG | DEBUG | INFO | - |
| `com.swingtrade.broker` | DEBUG | DEBUG | INFO | - |
| `com.swingtrade.core` | INFO | DEBUG | INFO | - |
| `org.springframework.web` / `.security` | INFO | (unset) | INFO | - |
| `org.hibernate.SQL` | DEBUG | INFO | WARN | local: INFO |
| `org.hibernate.type.descriptor.sql` | TRACE | WARN | (unset -> base TRACE) | local: WARN |
| `org.flywaydb` | INFO | DEBUG | INFO | - |

Plus the quiet-logger table in Spring Boot's `defaults.xml` (applies to all profiles): Tomcat
`DigesterFactory`/`LifecycleBase` ERROR, `Http11NioProtocol` WARN, sshd `SecurityUtils` WARN,
`NioSelectorPool` WARN, Jetty `AbstractLifeCycle` ERROR, Hibernate validator `Version` WARN,
Spring `actuator.endpoint.jmx` WARN. The `application-test.properties` (test scope) sets
`com.swingtrade` DEBUG, `springframework.test.context` DEBUG, `testcontainers` INFO.

## 6. Rolling, retention, paths

- `FILE` `RollingFileAppender`: `<file>${LOG_FILE}` + `SizeAndTimeBasedRollingPolicy` with
  `fileNamePattern ${LOG_FILE}.%d{yyyy-MM-dd}.%i.log`, `maxFileSize 50MB`, `maxHistory 7`,
  `totalSizeCap 2GB` (`logback-spring.xml:12-18`). Logback compresses rolled files only when the
  pattern ends in `.gz`; it ends in `.log`, so rolled files are **uncompressed** - confirmed on
  the node (`stage.log.2026-10-09.0.log`, no `.gz`).
- Stage on-disk reality (pi-node, 2026-10-16): 7 rolled files (Oct 9-15) plus the active file in
  the `stage_logs` named volume, sizes 75 KB to 1.1 MB per day at current stage traffic.
- The dead `max-size=100MB` / `max-history=30` stage properties (§2) mean the hard-coded 2 GB
  total cap and the 7-day history window are the only real bounds; at ~1 MB/day stage traffic the
  7-day window binds first (~7-11 MB total on disk) and the 2 GB cap is effectively unreachable
  at current volumes.
- Docker's `json-file` driver for the stage container has **no** `logging:` config in
  `docker-compose.infra-stage.yml`, so the console copy in the container log is unbounded
  (no `max-size`/`max-file`). The `stage` profile additionally sets
  `logging.pattern.console` (no colors), so `docker logs` output is plain.
- The legacy host-run path (`infra/legacy/github-actions-deploy-main-self-hosted.yml`) wrote
  `logging.file.name=/home/dietpi/swing-trade/logs/main.log`; that workflow is archived, and
  `docs/infra/ci-cd-deployment.md` known-issue 3 records the directory move. It is not a live
  consumer.

## 7. Consumer inventory for `logs/*.log` and console output

| # | Consumer | Consumes | Configuration location | Evidence / state |
|---|---|---|---|---|
| 1 | promtail (`pi-promtail`, pi-stack on pi-node) | **container stdout only**, via `docker_sd_configs` against `/var/run/docker.sock` for **all** containers; labels `job=docker`, `container_name`, `service_name` (both from the container name); **no file paths, no `pipeline_stages`, no level parsing** | node-local `/home/dietpi/pi-stack/promtail/promtail-config.yml` (a `.bak-20260828` sits beside it) - **not in this repo** | confirmed live: Loki label values include `swing-trade-stage-api`; a 7-day `{container_name="swing-trade-stage-api"}` query returns one stream of clean console-pattern lines. The `FILE` appender file is **not** scraped |
| 2 | Loki (`pi-loki`, pi-stack) | receives the promtail push to `http://loki:3100/loki/api/v1/push`; schema v13/tsdb, filesystem store in the `loki_data` volume | node-local `/home/dietpi/pi-stack/loki/config.yml` | confirmed: `retention_period: 168h` with `retention_enabled: true`; port published on the tailnet IP only (100.104.74.58:3100) |
| 3 | Grafana (`pi-grafana`) | a `Loki` datasource is provisioned (`/home/dietpi/pi-stack/grafana/provisioning/datasources/datasources.yml`, uid `loki`, url `http://loki:3100`) | node-local provisioning + repo `infra/monitoring/provisioning/dashboards/swingtrade.json` | the repo dashboard (15 panels) is **100% Prometheus; zero panels query Loki** - nothing dashboard-side depends on the log format |
| 4 | `dev-stack.sh logs-json` (local dev) | `tail -n 50 backend/logs/swing-trade-local.log \| jq '.' 2>/dev/null \|\| tail -n 50` (`dev-stack.sh:265-271`) | in-repo script | written expecting **one JSON object per line**; with the current non-JSON pattern `jq` fails and the raw `tail` fallback runs, so today it is a plain tail with a dead JSON attempt |
| 5 | `dev-stack.sh start` / `live-logs` (local dev) | console stdout redirected to `backend/logs/dev-stack-api-console.log` (`dev-stack.sh:116`); `live-logs` tails the two console-redirect files (`:209-216`) | in-repo script | human triage; also `frontend-logs`/`logs` commands for the infra containers |
| 6 | humans, stage | `docker --context pi-node logs -f swing-trade-stage-api` (the "Useful commands" block in `dev-stack.sh:stage`) and `docker exec ... tail /var/log/swing-trade/stage.log` | - | the only live reader of the FILE appender file is people |
| 7 | legacy GitHub Actions (archived) | `cat /home/dietpi/swing-trade/logs/main-start.log` of a host-run jar | `infra/legacy/github-actions-deploy-main-self-hosted.yml:64-83` | not a live path; stage deploys go through `dev-stack.sh stage` |

No dashboard source file, e2e spec, or in-repo script parses the log file or its MDC fields
(grep of `dashboard/src`, `e2e/`, `bin/`, `infra/` found none).

## 8. Findings that constrain the Go port

1. **The only monitored log path is container stdout.** Promtail uses `docker_sd` with no file
   paths and no pipeline stages, so Loki sees exactly the console appender's lines, one Loki
   entry per physical line, with three labels. The plan's "Logs: JSON" decision is compatible:
   promtail will keep shipping one JSON object per line untouched, and Loki's datasource can
   parse it. Nothing in the monitoring stack must be edited for the switch - except that the
   promtail config itself is **node-local and unversioned**, so any later change to it (e.g.
   adding a structured-metadata stage) is a manual step on pi-node during G8.
2. **Loki stream identity is the container name.** The stream label is
   `container_name=swing-trade-stage-api` (plus `service_name` of the same value). If the Go
   replacement runs under a different container name at G8, dashboards or ad-hoc queries
   filtering on the old name will go quiet after the 168 h retention. Keeping the same container
   name (or adding a relabel) is a cutover checklist item.
3. **The file-format quirk is a bug, not a contract.** The mid-pattern `%n` plus missing trailing
   newline (finding in §3) makes every file line a `<empty MDC stub><next event>` merge. No
   consumer reads the file, and the MDC fields are always empty in production, so the Go port
   should **not** reproduce this. Phase 1 step 5 (`log/slog` JSON handler with `traceId`,
   `symbol`, `module`) is the correct replacement: one JSON object per line for both stdout and
   file, with the three correlation fields as real JSON fields (populated for request work via
   the request middleware, absent or `""` for jobs - strictly more information than today).
4. **Thread naming is load-bearing for human triage.** Job-orchestrator, monthly-report, and
   scheduler pool names (`job-orchestrator-476`, `monthly-report-2`) appear in both console and
   file lines and are how operators tell which job emitted what. The Go port's goroutine/task
   naming should preserve comparable, stable names (the job orchestrator pool naming in
   particular), since that is the de-facto per-job correlation.
5. **Second-precision console vs millisecond file.** The console pattern is second precision
   (`%d{yyyy-MM-dd HH:mm:ss}`), the file pattern is millisecond. Loki entry timestamps come from
   promtail's wall clock, not the text. The Go JSON handler should use the same precision
   convention per sink (RFC 3339 with at least second precision) and keep `TZ=Asia/Kolkata`
   semantics - today's text stamps are IST, so switching to UTC text without a zone marker would
   silently shift every historical comparison.
6. **The stage file sink must keep working for the `docker exec tail` habit.** Keep the same
   path (`/var/log/swing-trade`), a named or bind volume, IST timestamps, and a sane rotation
   (the Go `lumberjack`-style equivalent of 50 MB / 7 / 2 GB is fine; the dead 100 MB / 30
   properties need no parity). Nothing else reads it.
7. **Per-profile levels are a parity item.** Local runs everything in `com.swingtrade` at DEBUG
   plus `org.hibernate.type.descriptor.sql` at WARN; stage runs everything at INFO plus
   `org.hibernate.SQL` at WARN. The Go config table (Phase 1 step 4) should carry an equivalent
   per-environment level map, and the hibernate-SQL-DEBUG base default is a local-only
   SQL-storm risk the Go port should not inherit.
8. **Double write and `includeCallerData` are removable.** Every event is formatted for both the
   console and file appenders, and `includeCallerData=true` captures a stack per event for a
   pattern that never prints the caller. In a 1 GiB / 1.5 CPU stage budget this is free
   performance the Go port (single JSON line, no caller capture) gets for nothing.
9. **`dev-stack.sh logs-json` will start working after the port.** Its `jq '.'` attempt exists
   today as a dead path; once logs are one-JSON-object-per-line it silently becomes the intended
   pretty-printer. No change needed, but it is a useful post-cutover sanity probe.
10. **The unbounded docker json-file log is pre-existing.** No `logging:` section in the stage
    compose; the console stream (which also feeds promtail) grows without a max-size cap on the
    node disk. The Go container should carry an explicit `max-size`/`max-file` `logging:` block
    at G8, or inherit the same unbounded behaviour.
11. **No structured-format or correlation hooks are in use.** `logging.structured.format.*` and
    `LOG_CORRELATION_PATTERN` are unset everywhere, and `CorrelationIdConverter` is registered
    by `defaults.xml` but never referenced - there is no hidden structured channel to preserve.
12. **Documentation drift.** `docs/infra/ci-cd-deployment.md` known-issue 3 still describes the
    legacy host-side log directory, and the promtail/loki configs that the
    `docs/plans/monitoring-stack.md` plan references live only on the node. The Go cutover
    (G8) should either version the node-local promtail/loki configs into `infra/monitoring/` or
    record the manual-copy step in the runbook, or the unversioned-config dependency will
    outlive the Java app.

## 9. Runtime confirmation (done, pi-node, 2026-10-16, all read-only)

- Read `/home/dietpi/pi-stack/promtail/promtail-config.yml` and its `.bak-20260828` (the only
  diff: the added `service_name` relabel) - `docker_sd`, no file paths, no pipeline stages.
- Queried Loki (`http://100.104.74.58:3100/loki/api/v1/query_range`, 7-day window): one stream
  `{container_name="swing-trade-stage-api", job="docker", service_name="swing-trade-stage-api"}`
  with clean console-pattern lines; label values list shows the stream has been flowing.
- `docker exec swing-trade-stage-api ls /var/log/swing-trade` + `tail ... | cat -A` +
  `grep -c '^ traceId='`: 7 rolled uncompressed files, the merged-stub layout of §3, 495 stub
  lines all with empty MDC values.
- Read `/home/dietpi/pi-stack/loki/config.yml`: `retention_period: 168h`, `retention_enabled:
  true`, single instance, filesystem store.
- Read `/home/dietpi/pi-stack/grafana/provisioning/datasources/datasources.yml`: Loki datasource
  present; repo `swingtrade.json` dashboard contains no Loki queries.

No containers were started and no node state was modified.
