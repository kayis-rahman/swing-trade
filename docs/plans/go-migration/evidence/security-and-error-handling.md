# API boundary, CORS, trace and error-handling inventory (Phase 0, step 12 cross-cutting)

Status: static characterization complete for the six cross-cutting concerns that apply to **every** endpoint rather than
to one handler. These are the rows the Go port must reproduce before any endpoint row can be closed, because they
change the observable response of all 156 live endpoints. Runtime confirmation (header and CORS behaviour against the
stage API on pi-node) is still pending Pi access — §9 lists the four checks.

Derived by: full read of `backend/api/src/main/java/com/swingtrade/api/config/`
(`ApiSecurityWebConfig`, `ApiSecurityInterceptor`, `CorsConfig`, `TraceIdFilter`, `GlobalExceptionHandler`,
`OpenApiConfig`), `backend/api/src/main/java/com/swingtrade/api/metrics/ApiRequestMetricsFilter.java`, the
Security / CORS / resilience4j / management blocks of `backend/api/src/main/resources/application.properties`,
`application-stage.properties` and `application-local.properties`, `backend/api/src/main/resources/logback-spring.xml`,
the deployed `infra/nginx/dashboard-nginx/default.conf` and `infra/docker-compose.infra-stage.yml`, plus the Java tests
`ApiSecurityInterceptorTest` and `GlobalExceptionHandlerTest` and the `@Disabled` `ErrorHandlingTest`. No product code
was changed.

## 1. Summary

| # | Concern | Java implementation | Default (dev) | Stage | Go target |
|---|---|---|---|---|---|
| 1 | Admin API-key boundary | `ApiSecurityInterceptor` on `/api/**` (`ApiSecurityWebConfig.java:19`) | off (`API_KEY_AUTH_ENABLED:false`) | **on** (`application-stage.properties:168`, compose requires `API_KEY`) | `api/config/api_security_interceptor.go` |
| 2 | Public per-client fixed-window rate limit | same interceptor, `allowPublicRequest` (`ApiSecurityInterceptor.java:88-107`) | 100/min per `remoteAddr` | 100/min per container IP (see finding F6) | same file |
| 3 | Per-endpoint Resilience4j rate limits | `@RateLimiter` on 8 handler methods + `application.properties:356-366` | 5/min per instance, fail fast | same | `api/config/resilience_rate_limiter.go` + middleware |
| 4 | CORS | `CorsFilter` bean (`CorsConfig.java`) on `/api/**` **and** a second, divergent `spring.web.cors.*` configuration | bean: 11 origins (both profiles) vs `spring.web.cors`: 2 origins | bean: 11 origins vs `spring.web.cors`: 3 origins (neither list matches the other) | `api/config/cors.go` |
| 5 | Trace / log correlation | `TraceIdFilter` → MDC only | on | on | `api/config/trace_id_filter.go` |
| 6 | Global error envelope | `GlobalExceptionHandler` + `ErrorResponse` | on | on | `api/config/global_exception_handler.go` |

Rows F-API-SEC-01…06 in the ledger track the port.

## 2. Request pipeline

Two independent layers sit in front of every controller, and they behave differently on rejection:

1. **Servlet filters** (all `@Component` / `@Bean` filters, order not declared):
   `CorsFilter` (`CorsConfig`) → `TraceIdFilter` → `ApiRequestMetricsFilter`
   (`OncePerRequestFilter`, records `api.request.duration` with `method`+`status` in a `finally`, so rejected requests
   are measured too — ledger F-OBS-28).
2. **`HandlerInterceptor`** `ApiSecurityInterceptor`, registered on `/api/**` only
   ([ApiSecurityWebConfig.java:19](../../../../backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityWebConfig.java#L19)).
   Filters always run before interceptors, so a preflight answered by `CorsFilter` never reaches the interceptor.

Because the interceptor is bound to `/api/**` and nothing else:

- `/actuator/**` (health, metrics, prometheus, flyway) is **unauthenticated and unbounded** — the Prometheus scrape
  target on stage depends on this (see [metrics-inventory.md §2](metrics-inventory.md)).
- Static assets, springdoc (`springdoc.api-docs.enabled=false` in
  `application-stage.properties:191-192`) and Spring Boot's own `/error` path sit outside `/api/**`, so none of them is
  ever subject to the admin key check.
- `/fyers/**` is **not** under `/api/**`, so it has no key check and no rate-limit budget either. That nginx block
  (`infra/nginx/dashboard-nginx/default.conf:53-59`) is dead configuration: the Fyers controller is mapped at
  `/api/fyers` (`FyersAuthController.java:26`), so backend `/fyers/**` has no mapping and answers 404. It is recorded
  here because a Go port that mirrors the *nginx* route table instead of the *interceptor* boundary would create a
  wrongly ungated path; the boundary to copy is `/api/**`.

**Go requirement**: reproduce the split exactly — key + public budget on `/api/**`, open actuator — or either the
scrape breaks (if the Go port gates `/actuator`) or the surface silently widens (if it gates nothing).

## 3. F-API-SEC-01/02 — the API-key boundary and the public rate window

Both live in [`ApiSecurityInterceptor`](../../../../backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityInterceptor.java).
One `preHandle` (`ApiSecurityInterceptor.java:63-71`) decides:

```java
if ("OPTIONS".equalsIgnoreCase(request.getMethod())) return true;   // preflight always passes
if (isAdminPath(request)) return authorizeAdmin(request, response); // key check, no rate-limit headers
return allowPublicRequest(request, response);                      // window budget, no key check
```

### 3.1 Properties

| Property | Dev default (`application.properties:289-291`) | Stage (`application-stage.properties:168-170`) | Meaning |
|---|---|---|---|
| `app.security.api-key.enabled` | `${API_KEY_AUTH_ENABLED:false}` | `${API_KEY_AUTH_ENABLED:true}` | enables the admin key check |
| `app.security.api-key` | `${API_KEY:}` | `${API_KEY:}` | shared secret, reads from env only |
| `app.security.public-rate-limit-per-minute` | `${API_PUBLIC_RATE_LIMIT_PER_MINUTE:100}` | same | budget, floored at 1 by `Math.max(1, publicLimit)` (`ApiSecurityInterceptor.java:57`) |

`infra/docker-compose.infra-stage.yml:43-44` sets `API_KEY_AUTH_ENABLED=true` and
`API_KEY=${API_KEY:?API_KEY must be set for the stage API}` — the stage API refuses to start without a key, and
`infra/env/.env.example:15-16` documents the pair. The Go port therefore runs with the admin boundary **on** on stage.

### 3.2 Admin path detection and the key check

`isAdminPath` (`ApiSecurityInterceptor.java:109-112`) is a literal test, not a regex:

```java
path.equals("/api/admin") || path.startsWith("/api/admin/")
```

`authorizeAdmin` (`ApiSecurityInterceptor.java:73-86`):

- key absent → **fail closed**: `401` with `WWW-Authenticate: X-API-Key` (`ApiSecurityInterceptor.java:83-84`). This is
  the case when `apiKeyEnabled=true` and the configured value is empty (`configuredApiKey.length > 0` guard at
  `ApiSecurityInterceptor.java:80`); pinned by
  `ApiSecurityInterceptorTest#failsClosedWhenEnforcementIsEnabledWithoutConfiguredKey`.
- comparison is `MessageDigest.isEqual` on UTF-8 bytes (constant-time for equal lengths; unequal lengths return early).
- header name is `X-API-Key` (`ApiSecurityInterceptor.java:33`), also declared as a springdoc security scheme
  (`OpenApiConfig.java:28-31`) — which never reaches a client because springdoc is disabled on stage.
- admin responses get **no** `X-RateLimit-*` headers: those are written only in `allowPublicRequest`.

### 3.3 The public window

`allowPublicRequest` (`ApiSecurityInterceptor.java:88-107`):

- Bucket key: `request.getRemoteAddr()` (`"unknown"` if null). **Nothing in the application reads
  `X-Forwarded-For`** — grep of `backend/**` finds no `forward-headers-strategy`, no `RemoteIpValve`/`server.tomcat.remoteip`
  setting and no header parsing, so with Spring Boot 3's default strategy (`none`) the bucket key is the socket peer.
  On the deployed topology (browser → nginx container → `api:8080`) that is the nginx container for **all** clients, so
  the 100/min public budget is one shared window per deployment, not per user. Runtime confirmation is in §9-F6.
- Window: fixed 1-minute window (`WINDOW = Duration.ofMinutes(1)`, `ApiSecurityInterceptor.java:34`), no sliding window,
  no warm-up: the first request in a window starts it, and the window is replaced when
  `!now.isBefore(current.startedAt.plus(WINDOW))`.
- Headers are written **before** the allow/reject decision (`ApiSecurityInterceptor.java:97-98`), so every non-admin,
  non-OPTIONS response — including the 429 itself — carries `X-RateLimit-Limit: <publicLimit>` and
  `X-RateLimit-Remaining: max(0, limit - count)`.
- Over budget: `api.rate.limit.hits` counter incremented (ledger F-OBS-35), status `429`, `Retry-After:
  max(1, seconds to window end)` (`ApiSecurityInterceptor.java:102-105`), and the interceptor returns `false`, which
  leaves the response with **status 429 and an empty body** — no `ErrorResponse`, no `ApiResponse`.
- The `windows` map is a plain `ConcurrentHashMap` (`ApiSecurityInterceptor.java:40`) that is never evicted; with a
  client-derived key it grows one entry per distinct address forever. On stage it has exactly one entry, so the Pi is
  safe today, but the Go port must not copy an unbounded map if bucket keying is ever fixed.
- `ApiSecurityInterceptorTest#limitsPublicRequestsAndExposesHitMetric` pins the boundary: with a limit of 2 the third
  request in the same fixed-clock window returns 429, carries `Retry-After` and increments the counter.
- `ApiSecurityInterceptorTest#optionsRequestsBypassApiBoundaryForCorsPreflight` pins that OPTIONS bypasses the key
  check **even on an admin path**.

### 3.4 Per-endpoint token gate (not the interceptor)

`POST /api/ingestion/reconcile` has its own optional gate: `X-Reconcile-Token` must equal
`ingestion.reconcile.apply-token` when that property is non-blank, plus `apply=true` plus a single active watchlist
symbol; mismatches return `403` with an `ApiResponse.error` body
([IngestionController.java:235-248](../../../../backend/api/src/main/java/com/swingtrade/api/controller/IngestionController.java#L235-L248)).
A blank configured token means the check passes — the same fail-open shape as the interceptor with key enforcement
disabled, but the opposite polarity from the admin boundary's fail-closed form.

## 4. F-API-SEC-03 — Resilience4j per-endpoint rate limits

A second, independent limiter layer sits inside the handler call, so it fires only for authenticated-through requests
and produces a **different** 429 shape.

| Instance | Limit | Refresh | Timeout | Annotated methods |
|---|---|---|---|---|
| `candidateScan` | 5 | 1m | 0 (fail fast) | `CandidateScanController:49` (start scan), `IntradayPaperScanController:25` |
| `llmAnalysis` | 5 | 1m | 0 | `AnalysisController:27`, `AnalysisOrchestrationController:56` |
| `sentimentAnalysis` | 5 | 1m | 0 | `SentimentApiController:160,206,214` (analyse, evaluate/trigger, pdf extract) |

Source: `application.properties:356-366`, with the comment "Fail fast (`timeout-duration=0`) rather than queueing
callers". `resilience4j.circbreaker.enabled=false`, `bulkhead.enabled=false`, `timelimiter.enabled=false`
(`application.properties:340-342`) because those instances are built programmatically in `Resilience4jConfig` (see
[metrics-inventory.md](metrics-inventory.md) §5/F-OBS-P10), but the **rate limiter** instances are the auto-configured
ones created from these properties — that is why `resilience4j.ratelimiter` has no matching bean in `Resilience4jConfig`.

Rejection: `RequestNotPermitted` is mapped by
[`GlobalExceptionHandler.handleRequestNotPermitted`](../../../../backend/api/src/main/java/com/swingtrade/api/config/GlobalExceptionHandler.java#L228-L243)
to `429` with body `{"status":429,"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests. Please try again shortly.",…}`

**Two distinguishable 429s exist and the Go port must keep both**: interceptor rejections are empty-bodied with
`Retry-After` and `X-RateLimit-*`, Resilience4j rejections are `ErrorResponse`-bodied with `code:
RATE_LIMIT_EXCEEDED` and no `Retry-After`. The dashboard only distinguishes them by response body
(`dashboard/src/api/shared.ts:118` maps status 429 to a `rate-limit` error kind and preserves `Retry-After` when
present, `dashboard/src/api/shared.test.ts:228-233`), so collapsing them is not a visible UI break — but tooling that
parses the body would see the difference.

## 5. F-API-SEC-04 — CORS

The `CorsFilter` bean ([`CorsConfig`](../../../../backend/api/src/main/java/com/swingtrade/api/config/CorsConfig.java))
is registered for `/api/**` only (`CorsConfig.java:50`) with:

- Origins (11): `http://localhost:3000`, `:3003`, `:3004`, `:3005`, `http://127.0.0.1:3003`, `:3005`,
  `http://localhost:8080`, `http://piworm.local:8081`, `:8082`, `http://127.0.0.1:8081`, `:8082`
  (`CorsConfig.java:23-35`). Note the stage dashboard is published on host port **8082**
  (`infra/docker-compose.infra-stage.yml:76`), which is in the list.
- Methods: `GET, POST, PUT, PATCH, DELETE, OPTIONS` (`CorsConfig.java:37`).
- Request headers: `Authorization`, `X-API-Key`, `Content-Type`, `Accept`, `X-Requested-With` (`CorsConfig.java:38-44`)
  — **not** `*`, so a browser preflight sending any other header (e.g. `Cache-Control`, `X-Requested-With` is listed)
  is rejected with 403 by the filter.
- Exposed headers: `Authorization` only (`CorsConfig.java:45`) — `X-RateLimit-Remaining`, `Retry-After`,
  `x-correlation-id` and friends are **not** exposed, so browser JavaScript cannot read the rate-limit state; only
  curl/Postman can.
- `allowCredentials=true`, `maxAge=3600` (`CorsConfig.java:46-47`) — legal only because the origin list is explicit.

A second, divergent CORS configuration is also active: `spring.web.cors.*`
(`application.properties:297-299` dev, `application-stage.properties:161-164` stage) uses
`allowed-headers=*`, `allowed-credentials=true` (stage) and a different origin list (dev 2 origins, stage
`localhost:3000,piworm.local:8081,127.0.0.1:8081`). Because a `CorsFilter` bean is present, Boot documents
`spring.web.cors` as the *alternative* to it, and the two lists do not agree (the property list omits
`piworm.local:8082`, the actual stage dashboard origin, while the bean includes it). **Which configuration wins for
`/api/**` on stage is unverified** (§9-F4): if the properties win, the dashboard's cross-origin calls fail because
`http://piworm.local:8082` is not allowed. Either way, the safe port target is the `CorsConfig` list, and the Go port
must drop the `spring.web.cors` properties rather than port both.

CORS is not idle even though the browser uses one origin: the Vite dev proxy **injects**
`Origin: http://localhost:3003` on every proxied request
([vite.config.ts:20-23](../../../../dashboard/vite.config.ts#L20-L23)), and Playwright runs against
`http://127.0.0.1:3003` (`dashboard/playwright.config.ts:6`). Both origins are in the `CorsConfig` list, so the filter
runs on every dev and e2e request — dropping either origin breaks the dev stack and the Playwright suite, not just
cross-origin browsers. In production nginx does not set `Origin` for same-origin dashboard traffic
(`infra/nginx/dashboard-nginx/default.conf:42-50`), so on stage the filter is only exercised by direct API calls that
choose to send an `Origin`.

| Consumer | Origin presented to the backend | In the `CorsConfig` list? |
|---|---|---|
| Vite dev server (`/api` proxy) | `http://localhost:3003`, set explicitly at `vite.config.ts:22` | yes (`CorsConfig.java:25`) |
| Playwright e2e | `http://127.0.0.1:3003` (`playwright.config.ts:6`) | yes (`CorsConfig.java:28`) |
| Stage dashboard through nginx | none — same-origin, nginx sets no `Origin` (`default.conf:42-50`) | n/a |
| Direct API consumer | whatever it sends | yes, but only `http://localhost:8080`, `http://piworm.local:8081`, `http://piworm.local:8082`, `http://127.0.0.1:8081`, `http://127.0.0.1:8082` |

CORS also interacts with the interceptor: OPTIONS is exempt from the key check and the public budget
(`ApiSecurityInterceptor.java:64`), and `ApiSecurityInterceptorTest#optionsRequestsBypassApiBoundaryForCorsPreflight`
pins that.

## 6. F-API-SEC-05 — trace and log correlation

[`TraceIdFilter`](../../../../backend/api/src/main/java/com/swingtrade/api/config/TraceIdFilter.java) is an MDC
populator, not a correlation-header emitter:

- `traceId` = the first 8 chars of a fresh `UUID.randomUUID()` per request (`TraceIdFilter.java:32`).
- `module` = literal `"api"` (`TraceIdFilter.java:33`).
- `symbol` = the `symbol` query param, else the `code` query param, else — only when the URI contains `/signals/` or
  `/positions/` — the last `/`-separated segment of the URI (`TraceIdFilter.java:36-45`); blank values are dropped.
- `MDC.clear()` in a `finally` (`TraceIdFilter.java:53`) wipes every key on the request thread.

**Correction to an earlier iteration's claim:** no `x-correlation-id`, `x-request-id` or `request-id` response header
is emitted anywhere in the backend. `grep -rn "setHeader\|addHeader"` over `backend/api/src/main/java` returns only
the four interceptor headers in §3. Iteration 1 recorded this as emitted behaviour; it is not. The dashboard
*demands* it optimistically — `dashboard/src/api/shared.ts:198-200` reads `x-correlation-id`, then `x-request-id`,
then `request-id`, and falls back to `body.traceId` (`shared.ts:192`) — so the Go port may add one of those headers
(and `standards.md` already requires `traceId` in the JSON log line) but no consumer requires it.

Log format (`backend/api/src/main/resources/logback-spring.xml:20`): a text pattern
`%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n traceId=%X{traceId} symbol=%X{symbol} module=%X{module}`.
There is **no JSON encoder anywhere** (`grep -ri "logstash\|jsonencoder\|jsonlayout"` over `backend/*/src/main/resources`
finds nothing), so the captain's "Logs: JSON" decision in [README.md](../README.md) is a *change* for the Go port, not
parity. Files land in the `stage_logs` volume (`LOG_FILE=/var/log/swing-trade/stage.log`,
`infra/docker-compose.infra-stage.yml:47,50`), streamed to stdout as well. Consumers of the file are inventoried in
[the completed Phase 0 log-field inventory](log-fields.md).

## 7. F-API-SEC-06 — the global error envelope

[`GlobalExceptionHandler`](../../../../backend/api/src/main/java/com/swingtrade/api/config/GlobalExceptionHandler.java)
(`@RestControllerAdvice`) always answers with the same record
([`ErrorResponse`](../../../../backend/api/src/main/java/com/swingtrade/api/dto/ErrorResponse.java)):
`status`, `code`, `message`, `timestamp` (`LocalDateTime` — no offset, non-null), `path`
(`request.getDescription(false)` with the leading `uri=` stripped) and `fieldErrors`, which is **never null** because
the default constructor initialises it to an empty list — so even validation responses serialize `"fieldErrors":[]`
instead of omitting it. That is a golden-corpus detail (Phase 0 step 5) rather than a security one, but it is pinned
here because it is the only error shape the dashboard sees.

| Exception | Status | `code` | Body source |
|---|---|---|---|
| `AsyncRequestNotUsableException` (SSE client gone) | **204**, no body | — | `GlobalExceptionHandler.java:35-39` |
| `MethodArgumentNotValidException` | 400 | `VALIDATION_ERROR` | `Validation failed: <field>: <message>, …` (field defaults inside, `fieldErrors` still `[]`) |
| `HttpMessageNotReadableException` | 400 | `VALIDATION_ERROR` | fixed message `Request body is invalid or unreadable` |
| `MissingServletRequestParameterException` | 400 | `VALIDATION_ERROR` | `Missing required parameter: <name>` |
| `MethodArgumentTypeMismatchException` | 400 | `VALIDATION_ERROR` | `Invalid value for '<name>': expected <simple type>` |
| `IllegalArgumentException` | 400 | `VALIDATION_ERROR` | raw exception message — **leaks internal wording** |
| `OptimisticLockingFailureException` | 409 | `CONFLICT` | fixed message, `Retry-After` absent |
| `RequestNotPermitted` | 429 | `RATE_LIMIT_EXCEEDED` | fixed message |
| `NoResourceFoundException` | 404 | `NOT_FOUND` | fixed message |
| `HttpRequestMethodNotSupportedException` | 405 | `METHOD_NOT_ALLOWED` | raw Spring message |
| `ResponseStatusException` | resolved status (500 fallback) | `HttpStatus.name()` (e.g. `I_AM_A_TEAPOT`) | raw reason; 5xx logged at error with stack |
| anything else | 500 | `INTERNAL_ERROR` | fixed message, full stack to the log |

Test coverage: `GlobalExceptionHandlerTest` (`#testHandleTypeMismatchReturnsBadRequest`,
`#testHandleIllegalArgumentReturnsBadRequest`, `#testHandleUnreadableMessageReturnsBadRequest`,
`#testErrorResponseStaticFactoryMethods[WithPath]`, `#testGlobalExceptionHandlerInstantiates`) is runnable but shallow —
it calls handler methods directly and asserts status/body, never the exception dispatch. `ErrorHandlingTest` (35 methods
across 5 nested classes: `TradeRequestValidation`, `HttpStatusCodeTests`, `ErrorResponseFormatTests`,
`ContentTypeTests`, `AcceptHeaderTests`) is the real specification and is `@Disabled("Stale against current
controller/service behavior")` — it tests the deleted `TradingController` and asserts `415` for a missing
`Content-Type`, which has no handler in `GlobalExceptionHandler` (Spring's own `HttpMediaTypeNotSupportedException`
would answer 415 without the envelope). The Go rows must re-derive these cases rather than port the test.

Hand-built error bodies bypass the envelope entirely: `JobRunController`, `CandidateScanController`,
`AnalysisController` and `SignalController` gate-effectiveness return `ResponseEntity` with ad-hoc 400/404/409 maps
(see [endpoint-inventory.md §3](endpoint-inventory.md)); the Go port must keep those as-is for wire parity.

## 8. Findings that shape the Go port

1. **The boundary is `/api/**` and nothing else.** Actuator (including `/actuator/prometheus`) is open and unbounded;
   `/fyers/**` is outside both the key check and the rate limit. Gating everything would break the stage scrape.
2. **Admin detection is a literal prefix test** on `/api/admin` / `/api/admin/`, so a Go router must add the check for
   exactly those paths and not for a general `/api/admin*` pattern.
3. **The admin boundary fails closed** (`401` + `WWW-Authenticate: X-API-Key`) when enforcement is on with no key
   configured, while the reconcile token check **fails open**. Both polarities must be preserved.
4. **OPTIONS bypasses everything**, including the admin key check — the Go port needs the same short-circuit before
   its CORS and auth middleware.
5. **Two 429 shapes** (empty body + `Retry-After` from the interceptor; `ErrorResponse`/`RATE_LIMIT_EXCEEDED` from
   Resilience4j) with different rate-limit semantics (100/min per IP vs 5/min per endpoint instance).
6. **Rate-limit headers are emitted on every non-admin response, before the decision**, including on 429s, so a Go
   middleware must write them exactly as Spring does (`Remaining = max(0, limit - count)`).
7. **The per-client bucket key is the socket peer, not the client**: nothing in the app honours `X-Forwarded-For`,
   so on the deployed topology the 100/min budget is a single deployment-wide window. Fixing this in Go would be a
   behaviour change, not parity.
8. **CORS has two divergent configurations** (a `CorsFilter` bean with 11 origins, and `spring.web.cors.*` with 2-3
   different origins and `allowed-headers=*`). Port the bean's list; drop the properties. The property list does not
   include the real stage origin `http://piworm.local:8082`, and two entries of the bean list
   (`http://localhost:3003`, `http://127.0.0.1:3003`) are load-bearing for the dev stack and the Playwright suite
   because the Vite proxy forces an `Origin` (see §5.1).
9. **No correlation response header exists** — earlier evidence claimed otherwise. `traceId`/`symbol`/`module` are log
   fields only. The dashboard reads the headers opportunistically and falls back to a body field, so adding a header
   is safe but optional.
10. **Logs are text with trailing MDC**, while the plan's decision is JSON logs — a deliberate parity break for the Go
    port, recorded here so it is not mistaken for an accident when the log shape changes at cutover.
11. **`fieldErrors: []` is always serialized** and `timestamp` is always present, so error goldens must include both.
12. **nginx rewrites two dashboard health paths**: `location = /api/health` and `= /api/health/full` in
    `infra/nginx/dashboard-nginx/default.conf:25-39` proxy to the backend `/actuator/health`, not to
    `HealthController`. On stage the browser never reaches `HealthController#health` / `#full`; the other
    `/api/health/*` routes (`details`, `database`, `market-data`, `llm`) do reach it through `location /api/`. The Go
    port must keep answering both paths, and the read-edge phase must not assume the dashboard exercises the
    controller's `/api/health`.
13. **`/api/actuator/` is proxied too** (`default.conf:14-22`, to `/actuator/`), so the boundary rule "open actuator"
    applies to a path the dashboard actually uses; a Go port that moved actuator under `/api/**` would start requiring
    a key for it.

## 9. Runtime confirmation still pending (needs stage)

| ID | Check |
|---|---|
| F4 | From a browser origin of `http://piworm.local:8082`, send an OPTIONS preflight to `/api/signals` with `curl -i -X OPTIONS -H "Origin: http://piworm.local:8082" -H "Access-Control-Request-Method: POST" http://piworm.local:8082/api/signals` and check which origin and header list comes back, including whether `X-API-Key` is allowed. |
| F5 | `curl -i -H "X-API-Key: <stage key>" http://piworm.local:8081/api/admin/health` from inside the compose network: expect 200; without the header expect 401 + `WWW-Authenticate: X-API-Key` and an empty body. |
| F6 | Hit a public path twice from two different client IPs and read `X-RateLimit-Remaining`: does it count down per client (X-Forwarded-For honoured) or per deployment (single nginx window)? |
| F7 | Trigger a Resilience4j 429 (6th `POST /api/analysis/analyze` within a minute) and confirm the body is the `RATE_LIMIT_EXCEEDED` `ErrorResponse` with no `Retry-After`, versus the interceptor's empty-bodied 429. |
