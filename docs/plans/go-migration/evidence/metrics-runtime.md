# Phase 0 metrics runtime confirmation

Read-only checks performed on 2026-10-17T12:47:03Z against the existing stage API and the existing read-only
SSH tunnel to Prometheus. No containers, services, configuration, or data were changed.

## Commands

```sh
curl -fsS http://127.0.0.1:19090/api/v1/targets
curl -fsS http://127.0.0.1:19090/api/v1/rules
curl -fsS 'http://127.0.0.1:19090/api/v1/series?match%5B%5D=%7Bjob%3D%22swing-trade-stage%22%7D'
curl -fsS http://piworm.local:8081/actuator/prometheus
docker --context pi-node exec swing-trade-stage-api sh -c 'grep -E "^(VmRSS|VmHWM|VmSize):" /proc/1/status'
curl -fsS 'http://127.0.0.1:19090/api/v1/query?query=histogram_quantile%280.95%2C+sum+by+%28le%29+%28rate%28http_server_requests_seconds_bucket%7Bjob%3D%22swing-trade-stage%22%7D%5B5m%5D%29%29%29'
```

The queries return JSON/text; raw outputs were retained outside the repository in a task-local evidence directory.

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
- A read-only process snapshot reported `VmRSS=703488 kB`, `VmHWM=842000 kB`, and `VmSize=2514336 kB` for
  PID 1 in the existing stage API container. Actuator reported `application_started_time_seconds=55.78` and
  `application_ready_time_seconds=66.478`, with heap pools totaling about 338.6 MB at scrape time. A Prometheus
  5-minute rate query at 2026-10-17T12:55:14Z returned aggregate HTTP request quantiles p50 `0.0146801 s`,
  p95 `0.0221599 s`, and p99 `0.0263402 s`; `process_cpu_usage` was `0.00588235` (fraction). These are
  observational stage values, not a controlled Phase 0 baseline: request volume/workload was not controlled,
  the Java build is not matched to a Go binary, and `docker stats` returned `0B / 0B` from this remote context,
  so it was not used for the RSS value. No full-universe backtest was run on Pi; it would not be read-only.

## Raw evidence checksums

SHA-256 checksums of the retained responses (which contain metric values and labels, but no credentials):

| Artifact | SHA-256 |
|---|---|
| `prometheus-targets.json` | `ab2ceaa1c101ae175e9c60c2b4f48cbc95798cf7d5e865bd9daa26a8f93ac0b4` |
| `prometheus-rules.json` | `992e1ec3162e1f708f94bb155f4cf306c499145c541da42ffe780446b852c6d6` |
| `stage-job-series.json` | `8d5186f86a699f8ede39fe2a60392cfc82a08aa956789ff843c166dcfb51a1ec` |
| `stage-metrics.prom` | `fb5ddc0765c8ee5389ab2fc9b5c5072febd894b661781f7ee94f802d9b5e8bc7` |
| `request_p50.json` | `633733ffa2e6cbd4e8e11f7ddc147ea167a6e7ca3aee3ab9eeede23099f382d3` |
| `request_p95.json` | `cc7d3ea28367891dabc2f211ddd9b1ed5bdbe57c0a0b13fe820424ad0e574c28` |
| `request_p99.json` | `2b02b34776531f94722be128a493f71b17042523de49709d7589d7ea64f85401` |
| `process_cpu_usage.json` | `e7d41e8c3c2929850a93baae05069697b25e9b57a4fc90c7c8a6e4316908fb16` |
| `process_cpu_time.json` | `60deb7d9e2367ec7667bdc9326220be7040324819d6c359ed8a3dad95b8648db` |
