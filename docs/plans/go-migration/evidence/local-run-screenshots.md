# Phase 1 local-run screenshot evidence

Captured 2026-10-19 on loopback. These artifacts are local-run proof for PR #139; they do not indicate production readiness or authorize stage/Pi deployment.

## Combined dashboard and service run

The dashboard was served on `127.0.0.1:5179`. The existing Java backend was served on `127.0.0.1:8080`, and the no-traffic Go skeleton was served independently on `127.0.0.1:8081`. Java health, Go health, and Go Prometheus requests returned HTTP 200. Browser page errors: none.

![Dashboard loaded with the Java backend and Go service healthy on separate loopback ports; dashboard data is synthetic/empty local state.](local-run/dashboard-java-go-healthy.png)

The dashboard remained connected to the Java backend. Go was running independently; the dashboard was not routed through or switched to Go. This demonstrates coexistence and separation, not Go feature integration with dashboard endpoints.

## Real-binary Go route walkthrough

The route walkthrough used the real Go binary. Each request returned HTTP 200. The corresponding browser captures are included below, along with a video of the route walkthrough.

| Route | Result | Capture |
|---|---:|---|
| `/api/health` | HTTP 200 | ![Go API health route returned HTTP 200](local-run/go-api-health-route.png) |
| `/api/admin/health` | HTTP 200 | ![Go admin health route returned HTTP 200](local-run/go-admin-health-route.png) |
| `/actuator/health` | HTTP 200 | ![Go actuator health route returned HTTP 200](local-run/go-actuator-health-route.png) |
| `/actuator/health/liveness` | HTTP 200 | ![Go liveness route returned HTTP 200](local-run/go-liveness-route.png) |
| `/actuator/health/readiness` | HTTP 200 | ![Go readiness route returned HTTP 200](local-run/go-readiness-route.png) |
| `/actuator/prometheus` | HTTP 200 | ![Go Prometheus metrics route returned HTTP 200 with exported metrics](local-run/go-prometheus-route.png) |

[View the real-binary Go route walkthrough video](local-run/go-route-walkthrough.webm).

## Rate-limit expiry on the real binary

With the public limit set to 100 requests per minute, 101 rapid requests from one loopback client produced 100 HTTP 200 responses followed by HTTP 429. After waiting 65 seconds, the next request returned HTTP 200 with `X-RateLimit-Limit: 100` and `X-RateLimit-Remaining: 99`. The standalone Go binary ran with no database configured.

![Real-binary rate-limit response and post-expiry reset results](local-run/go-rate-limit-eviction.png)

[View the rate-limit expiry video](local-run/go-rate-limit-eviction.webm).

## Local environment and safety

- Java used a new, empty disposable database on the existing local PostgreSQL instance. Flyway initialized that database; it was dropped after capture.
- Java startup was configured with loopback binding and its startup benchmark backfill, EOD ingestion, and schedulers disabled. The Go service also bound to loopback. No dev, stage, or Pi database or host was accessed in the successful screenshot run.
- During an earlier, preliminary startup attempt, Java's default benchmark backfill did issue Yahoo Finance requests and write fetched candles only to a throwaway local database. That database was dropped. The final screenshot run explicitly disabled the backfill. See [local boot diagnostic](local-run/boot-diagnostic.md) in the external task evidence for details.
- All services and the throwaway database were stopped/dropped after capture.

## Review coverage limitation

The no-mistakes review reported no findings in the final review pass, but reported 45 reviewable files unverified because of a review coverage limit. The remaining paths were lockfiles, module manifests/checksums, and planning/evidence documentation. This coverage limitation was recorded for the captain; it is not evidence that those paths received individual review.

## Boundaries

These screenshots prove only local startup and route responses. They do not show production traffic, Java replacement, API routing to Go, stage deployment, Pi measurements, or stage backup/restore. Those remain out of scope or deferred pending separate authorization.
