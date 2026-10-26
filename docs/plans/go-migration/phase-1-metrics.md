# Phase 1 Go platform metrics

The Phase 1 registry contains only platform-level request metrics. It does not claim parity with the 35 application meters or Spring Boot JVM/process/Hikari meters inventoried in [Phase 0](evidence/metrics-inventory.md); those require feature ports or runtime parity work and are not active in this no-traffic skeleton.

| Go Prometheus series | Micrometer source equivalent | Labels | Notes |
|---|---|---|---|
| `http_server_requests_seconds_count`, `_sum`, `_bucket` | `http.server.requests` timer | `application`, `method`, `uri`, `status`, `outcome`, `exception` | Histogram with stable route-pattern labels; URI values use path patterns, never raw query strings |
| `api_request_duration_seconds_count`, `_sum` | API request filter timer | `application`, `method`, `status` | Summary with no client-side quantiles or buckets |
| `api_rate_limit_hits_total` | `api.rate.limit.hits` counter | `application` | Increments only when the per-IP public limit rejects a request |

`application` is fixed to `swing-trade-api`, matching the Java Micrometer common tag. The unauthenticated scrape endpoint is `/actuator/prometheus`. Admin API-key protection applies only to `/api/admin/*`, and public rate-limit headers match the Java interceptor. The real-binary E2E asserts all three series families appear after requests.

This mapping is a local skeleton contract, not a stage scrape comparison. The Phase 0 runtime metrics capture and stage endpoint remain unchanged; no stage request is authorized in Phase 1.
