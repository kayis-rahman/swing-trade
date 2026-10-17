# Phase 0 metrics runtime confirmation

Read-only checks performed on 2026-10-17T12:47:03Z against the existing stage API and the existing read-only
SSH tunnel to Prometheus. No containers, services, configuration, or data were changed.

## Commands

```sh
curl -fsS http://127.0.0.1:19090/api/v1/targets
curl -fsS http://127.0.0.1:19090/api/v1/rules
curl -fsS 'http://127.0.0.1:19090/api/v1/series?match%5B%5D=%7Bjob%3D%22swing-trade-stage%22%7D'
curl -fsS http://piworm.local:8081/actuator/prometheus
```

The queries return JSON/text; raw outputs were retained outside the repository under
`/Users/kayisrahman/Documents/workspace/firstmate/data/swing-trade-go-migration-impl-v1/`.

## Results

- Prometheus API responses returned `status=success`.
- Target `swing-trade-stage` was `up`; scrape URL was
  `http://swing-trade-stage-api:8080/actuator/prometheus`; `lastError` was empty.
- The stage endpoint exported 146 metric families. Prometheus had 151 names for the stage scrape job; the
  five additional names were `scrape_duration_seconds`, `scrape_samples_post_metric_relabeling`,
  `scrape_samples_scraped`, `scrape_series_added`, and `up` (Prometheus scrape-health series).
- Two `resilience4j_` families were present: `resilience4j_ratelimiter_available_permissions` and
  `resilience4j_ratelimiter_waiting_threads`, each with `name` values `candidateScan`, `llmAnalysis`, and
  `sentimentAnalysis`. These are the three property-configured `@RateLimiter` instances in
  `application.properties` (the controllers use `@RateLimiter`); this explains the auto-bound metrics despite
  the separate programmatic circuit-breaker/bulkhead/time-limiter registries. Preserve both families and their
  `name` values in the Go parity target.
- Prometheus returned zero rule groups. Thus the `StrategySkipped` and `StageDegraded` alerts are not loaded
  in the queried Prometheus instance.

## Raw evidence checksums

SHA-256 checksums of the retained responses (which contain metric values and labels, but no credentials):

| Artifact | SHA-256 |
|---|---|
| `prometheus-targets.json` | `ab2ceaa1c101ae175e9c60c2b4f48cbc95798cf7d5e865bd9daa26a8f93ac0b4` |
| `prometheus-rules.json` | `992e1ec3162e1f708f94bb155f4cf306c499145c541da42ffe780446b852c6d6` |
| `stage-job-series.json` | `8d5186f86a699f8ede39fe2a60392cfc82a08aa956789ff843c166dcfb51a1ec` |
| `stage-metrics.prom` | `fb5ddc0765c8ee5389ab2fc9b5c5072febd894b661781f7ee94f802d9b5e8bc7` |
