# Golden corpus and wire-format evidence

Phase 0 steps 2 (OpenAPI export), 4 (golden request/response pairs) and 5 (time and
decimal formatting rules). Everything in this document was captured from a real run of
the API at HEAD (`15766cca`) against an isolated throwaway database on this machine, on
2026-10-15. No product code was modified.

| Artifact | What it is |
|---|---|
| [scripts/capture-goldens.py](scripts/capture-goldens.py) | the reproducible capture script |
| [goldens/openapi3.json](goldens/openapi3.json) | the live springdoc 3.1 export (142 paths, 109 schemas) |
| [goldens/index.md](goldens/index.md) | generated index of the 71 default-profile cases + the rate-limit probe |
| [goldens/cases/*.json](goldens/cases) | per-case request, status, full response headers and body |
| [goldens-security/](goldens-security) | the 5 security cases captured against an API-key-enabled instance |

## 0. How to reproduce

```bash
# 1. isolated throwaway Postgres (a local cluster, never the shared stage/dev DB)
initdb -D /tmp/throwaway/pgdata -U postgres --auth-local=trust --auth-host=trust --encoding=UTF8
pg_ctl -D /tmp/throwaway/pgdata -o "-p 55432 -h 127.0.0.1" -l /tmp/throwaway/pg.log start
psql -h 127.0.0.1 -p 55432 -U postgres -c "create role swingtrade_user login superuser password 'swingtrade_password'"
psql -h 127.0.0.1 -p 55432 -U postgres -c "create database swingtrade_golden owner swingtrade_user"

# 2. build the API jar offline (the Gradle cache is warm; no network needed).
#    NB: backend/gradle/wrapper/gradle-wrapper.jar is untracked+gitignored (*.jar) and is absent from a
#    fresh clone, so ./gradlew fails with "Unable to access jarfile" until it is restored from a
#    working checkout (e.g. ../swing-trade/backend/gradle/wrapper/). It was restored locally for this
#    run and is not part of any diff.
cd backend && ./gradlew :api:bootJar --offline -q

# 3. run it. The command below EXITS 1 at HEAD (finding 1) unless the Jackson 2 ObjectMapper bean is
#    supplied from outside the product tree; that external launcher is what produced this corpus.
#    Scheduling and the LLM are disabled and the LLM base URL points at a dead local port: without
#    llm.backend=local the local profile's default pi_ssh backend SSHes to the Pi and tries to
#    START llama-server (finding 7).
java -Duser.timezone=Asia/Kolkata -jar backend/api/build/libs/api.jar \
  --spring.profiles.active=local \
  --spring.datasource.url=jdbc:postgresql://127.0.0.1:55432/swingtrade_golden \
  --app.scheduling.enabled=false --llm.enabled=false --llm.backend=local \
  --llm.base-url=http://127.0.0.1:1/v1 --logging.file.name=/tmp/throwaway/app.log

# 4. capture
python3 docs/plans/go-migration/evidence/scripts/capture-goldens.py \
  --base http://127.0.0.1:8080 --out docs/plans/go-migration/evidence/goldens \
  --tag local-throwaway --rate-limit-probe 130

# 5. the security cases need a second instance with the gate on (as stage runs it)
java ... --server.port=8081 --app.security.api-key.enabled=true \
  --app.security.api-key=gnhf-test-key --app.security.public-rate-limit-per-minute=100000
python3 docs/plans/go-migration/evidence/scripts/capture-goldens.py --base http://127.0.0.1:8081 \
  --out .../goldens-security --tag local-throwaway-security --api-key gnhf-test-key --security
```

The script exits non-zero when a case does not return the status recorded here, so it doubles
as a regression check for the Go port.

## 1. BLOCKING FINDING: the API at HEAD does not start

`IntradayPaperController` requires a `com.fasterxml.jackson.databind.ObjectMapper` bean
(constructor parameter 4) that no Spring Boot 4.1.1 auto-configuration provides, so
`preInstantiateSingletons` aborts and the JVM exits with code 1 after a *silent* graceful
shutdown (Hikari + Tomcat stop, no `APPLICATION FAILED TO START` line, because the failure
is reported by Boot's `LoggingFailureAnalysisReporter` on the application logger and the
process is torn down before anything else useful is printed):

```text
UnsatisfiedDependencyException: Error creating bean with name 'intradayPaperController' ...
  Unsatisfied dependency expressed through constructor parameter 4:
  No qualifying bean of type 'com.fasterxml.jackson.databind.ObjectMapper' available
Caused by: NoSuchBeanDefinitionException: No qualifying bean of type
  'com.fasterxml.jackson.databind.ObjectMapper' available
```

- Introduced by `2ce79be0` (feat(api): operator-selected analysis scope and persisted intraday
  paper sessions, #136) — `git log -S "ObjectMapper objectMapper"` confirms the parameter is new there.
- Both Jackson 2 (2.21.5) and Jackson 3 (3.1.5) are on the runtime classpath, but Boot 4's
  `JacksonAutoConfiguration` (spring-boot-jackson 4.1.1) only registers the Jackson 3
  `tools.jackson.databind.ObjectMapper`/`JsonMapper`; there is no Jackson 2 bean source.
- The container that is running stage was built 2026-10-10 from `fe3cc3f1` and therefore
  predates #136, which is why stage still looks healthy. Any build from HEAD is dead on
  arrival: `dev-stack.sh` and `infra/Dockerfile` both start the same entry point.
- Tests do not catch it: `IntradayPaperControllerTest` instantiates the controller directly
  (per the iteration-1 inventory) and the three full-context `@SpringBootTest` classes are
  `@Disabled`, so nothing ever refreshes the context in CI.
- `SettingsAwareChatModels` (llm) also imports Jackson 2's ObjectMapper but builds its own
  static instance, so it is unaffected.

The capture above was produced by registering the missing bean from an external launcher
(`/tmp`, outside the product tree) — a read-only workaround for characterization only. For
Phase 0 to complete against HEAD, someone must land the product fix (inject nothing and build
the mapper where it is used, or use `Jackson2ObjectMapperBuilder`); this is a Java-side ticket,
recorded here because it gates every runtime evidence artifact in this plan.

## 2. Step 2: the live OpenAPI export

- Captured at `GET /api-docs` (springdoc 3.1.1, OpenAPI 3.1.0): **142 paths, 109 component schemas**.
- The path set is stable across two independent runs (unauthenticated and API-key-enabled) — schema
  and path membership identical, only key ordering differs.
- The static inventory from iteration 1 counted 156 live endpoint mappings across 28 controllers;
  the live export has **142 path keys carrying exactly 156 operations** (13 paths carry two or
  three methods: `candidate-scans`, `holidays`, `ingestion/reconcile`, `positions`,
  `settings/{discord,gpuhub,llm,scanning,trading}`, `strategy-configs`,
  `strategy-configs/{variantId}`, `watchlist`). The static count and the live operation count agree
  exactly, which cross-validates both artifacts: nothing live was missed by the static survey and
  springdoc exposes no route the survey did not.
- `springdoc.api-docs.enabled=false` in `application-stage.properties`, so the export only exists in
  the local profile. The Go port must decide whether to keep an OpenAPI artifact at all; nothing in
  stage consumes it today.
- No `/v3/api-docs` alias is mapped (the property `springdoc.api-docs.path=/api-docs` in the local
  profile is honoured), so any tooling that assumes the springdoc default will 404.

## 3. Step 5: time, decimal, enum and null formatting rules actually emitted

| Rule | Evidence (case) |
|---|---|
| `LocalDateTime` has **no zone offset** and **microsecond precision** (6 fractional digits) | `"timestamp":"2026-10-15T07:33:23.818683"` on every error envelope, `settings`, `positions-performance` |
| `LocalDate` is a bare `yyyy-MM-dd` | `"analyzedAt":"2026-10-15"`, `"date":"2026-10-15"` |
| **`BigDecimal` is emitted as a quoted JSON string, not a number** | `settings-trading`: `"trading.allocation_per_position":"100000"`, `"trading.initial_capital":"500000"`, `"trading.stop_loss":"5"`, `"trading.take_profit":"15"` — this contradicts the plan's "BigDecimal as number" assumption and is the single highest-risk parity rule for the Go port |
| `double`/`float` values are emitted as JSON numbers | `sentiment/RELIANCE/latest`: `"confidence":0.5` |
| `int`/`long` are unquoted numbers, ids included | `"id":12`, `"articleCount":10`, `"defaultValue":20` |
| Enums are upper-case strings | `"score":"NEUTRAL"`, `"type":"INT"` |
| A null field is **present with an explicit null**, never omitted | `"error":null`, `"signalType":null`, `"averageLoss":0` vs `"averageTrade":null` (zero vs absent aggregates differ) |
| Empty collections are `[]`, empty objects `{}` | `"content":[]`, `"details":{}` |
| Every response carries `Vary: Origin` | all cases |
| CSV export is `text/csv` and its header row has a trailing space | `data-export`: `symbol,date,open,high,low,close,volume␠` |
| Non-admin responses carry `X-RateLimit-Limit` / `X-RateLimit-Remaining`; admin ones do too | `goldens-security/cases/rate-limit-headers-on-success` |

## 4. Response envelope families (the wire shapes the Go port must reproduce)

Four families coexist; the dashboard branches per call site on `direct` vs `envelope`:

1. **`ErrorResponse`** — `{code, fieldErrors, message, path, status, timestamp}`, where `fieldErrors`
   is *always* present as `[]` and `timestamp` is always offset-less. Every 400/404/401 body.
2. **`ApiResponse<T>`** — `{data, error, success, timestamp}` with `error:null` on success
   (`settings`, `watchlist`, `sentiment/{symbol}/latest`, `admin/health`).
3. **Spring `Page`** — `{content, first, last, pageNumber, pageSize, totalElements, totalPages}`
   (`positions`). Note `first`/`last` are booleans even for an empty page and `pageNumber` is
   0-based, unlike the `page`/`size` query parameters it answers.
4. **Bare values** — a raw `[]` array (`signals/latest`) and a bare `Map`
   (`DELETE /api/signals/{symbol}` → `{"cleared":0,"symbol":"..."}`).

## 5. Status codes observed on an empty throwaway database

| Status | Trigger | Case |
|---|---|---|
| 200 | normal reads, empty collections, `DELETE` on a live symbol | most cases |
| 400 | missing required query parameter | `Missing required parameter: symbol` (`admin-data-status`), `from` (`gate-effectiveness`), `fromDate` (`data/export`) |
| 400 | missing required request parameter/body field | `POST /api/watchlist` with `{}` and with `{"foo":"bar"}` both → `Missing required parameter: symbol` (no `fieldErrors` are ever populated by this path) |
| 400 | range guard, not a validation annotation | `ingestion/reconcile?from=2026-01-01&to=2026-01-31` → `reconciliation range must be at most 10 calendar days`; a 5-day window returns 200 |
| 400 | unknown identifier | `strategy-configs/unknown-variant` → `Strategy variant not found: unknown-variant` (**400, not 404**) |
| 400 | reserved path segment on `DELETE /api/signals/{symbol}` | only `latest`, `date-range`, `scan` are reserved → bare `{"error":"'latest' is a reserved path segment, not a stock symbol"}` with **no** envelope. `generate` is *not* reserved and is answered by the `{symbol}` handler |
| 401 | admin endpoint without/with a wrong key | empty body, `WWW-Authenticate: X-API-Key`, `Connection: close` |
| 404 | unmapped path | `NOT_FOUND`, `The requested resource was not found`, `path` echoed |
| 429 | public fixed-window limiter | empty body, `Retry-After: 17`, `X-RateLimit-Limit: 100`, `X-RateLimit-Remaining: 0` |
| 500 | `/api/admin/data/status?symbol=RELIANCE&days=30` | **not** the `ErrorResponse` envelope: `{"data":null,"error":"Failed to get data status: null","success":false,"timestamp":...}` — an internal message leaks into a 500 body |

Not yet captured (still owed for step 4): a 409 conflict, a 422, and any golden that exercises a
populated money path. All of those need either a seeded database or the stage read path.

## 6. The 429 bucket is one deployment-wide counter, demonstrated live

The rate-limit probe fired requests at `/api/signals/latest` until the first rejection. It was
rejected on **attempt 33**, because the 71-case capture that ran immediately before it in the same
minute had already consumed 67 of the 100 requests: the fixed window is keyed on the remote address
only and counts every request regardless of path or method. The Go port must reproduce the shared
bucket (and the `Retry-After` value is the remaining window seconds, not a fixed retry delay).

## 7. Side effects a golden capture must avoid

- With the local profile default `llm.backend=pi_ssh`, `GET /api/signals/combined/{symbol}` calls
  `PiLlamaServerManager.ensureRunning`, which **SSHes to pi-node and tries to start llama-server**
  (`IllegalStateException: llama-server on Pi failed to become inference-ready within 45s`). Any
  characterization run must override `llm.backend` to a dead local URL first. This also means the
  local profile is not safe for unattended runs.
- The same endpoint fetches live Google News RSS (`NewsIngestionService`), so its body is
  network-dependent. On an unreachable LLM it degrades to keyword analysis and returns 200 with
  `{"signalType":null,"technicalAnalysis":null,"sentimentAnalysis":{...,"score":"NEUTRAL",
  "confidence":0.5,"articleCount":10,...}}` — that degradation path is itself a golden worth pinning.
- The throwaway database is **never empty**: `V1__swing_trade_schema.sql` seeds the watchlist (20
  NSE symbols, `AXISBANK`, `BHARTIARTL`, …), `app_settings`, the kill-switch row, the 2026 NSE
  holiday calendar and one paper-trading portfolio, so Flyway alone produces a populated database.
  Case bodies are therefore not byte-stable across runs; comparisons must normalise timestamps and
  ids, which is why the script records the raw body rather than asserting equality.
- The capture writes to the database (the `DELETE` probes and the sentiment writes above). Only the
  throwaway cluster was used; nothing was written to the dev or stage databases.

## 8. Remaining gaps in this deliverable

1. Golden vectors for money paths (positions, P&L, backtest metrics) need a seeded database.
2. SSE event streams must be captured with `curl -N` as raw text; the three endpoints are
   `POST /api/signals/generate-all/stream`, `GET /api/candidate-scans/{runId}/stream` and
   `POST /api/analysis/run-full`.
3. The runtime confirmation half of step 8 (metrics inventory) is now possible: `/actuator/prometheus`
   answered 200 with `Accept: */*` on the throwaway run, and the corpus records its first bytes.
   Note it answers **500 `No acceptable representation`** (via `GlobalExceptionHandler`) when the
   client sends `Accept: application/json`, which is what a naive JSON client would send.
4. The scheduling-off runtime test (step 12a) still needs a run with the flag on to diff job logs.

## 9. Schema snapshot (step 7, first half)

`pg_dump --schema-only` against the throwaway cluster is committed as
[schema-snapshot.sql](schema-snapshot.sql) — the exact schema the 53 checked-in Flyway migrations
produce, 2 990 lines.

| Fact | Value | Why it matters for the port |
|---|---|---|
| `CREATE TABLE` statements | 42 (41 application tables + `flyway_schema_history`) | the plan's area table claims 35 entities and 37 repositories; the real table count needs no interpretation, so the entity/repository rows in the ledger can now be checked against a fixed number |
| indexes | 78 `CREATE INDEX`/`CREATE UNIQUE INDEX` | index names are the parity surface for query plans, and several of them are the idempotency guard named in the scheduler characterization |
| TimescaleDB / hypertables | none — no `CREATE EXTENSION`, no hypertable anywhere in the 53 migrations, and no `timescaledb` string in the migration tree | plain Postgres is sufficient; the "TimescaleDB-patched PostgreSQL" comment in `application.properties` is stale and the Go port needs no extension |
| Flyway rows | 53, `max(version) = 77` | the version numbers are **not contiguous**: the file set jumps `V1` → `V25` and skips `V71`, so the goose "no-op V76 baseline" from the cutover decision must preserve these 53 exact rows rather than renumbering, or the schema-history table stops matching the Java history |
| Seeded data in `V1__swing_trade_schema.sql` | watchlist (20 NSE symbols), `app_settings`, the single `kill_switch` row, the 2026 NSE holiday calendar, one paper-trading portfolio | a "fresh" database is never empty; every empty-collection golden in §5 is therefore a *seeded-empty* state, and the ownership sheet must treat these as write-once baseline rows |

Still owed for step 7: the table ownership sheet (writers, readers, retention, unique keys and the
ingestion idempotency proof), which is a source-analysis artifact and does not need a database.
