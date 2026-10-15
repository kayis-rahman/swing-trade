# Security-profile golden corpus (API-key-enabled instance)

Captured from a second instance started with `--app.security.api-key.enabled=true
--app.security.api-key=gnhf-test-key --app.security.public-rate-limit-per-minute=100000`.
Capture tag: `local-throwaway-security`. Base: `http://127.0.0.1:8081`.

Only the five security cases are kept here; the other 71 cases duplicated [../goldens/](../goldens) and were
removed. Reproduce with:

```bash
python3 scripts/capture-goldens.py --base http://127.0.0.1:8081 --out goldens-security \
  --tag local-throwaway-security --api-key gnhf-test-key --security
```

| case | method | path | status | expected | note |
|---|---|---|---|---|---|
| rate-limit-headers-on-success | GET | `/api/signals/latest` | 200 | 200 | successful response still carries X-RateLimit-Limit/Remaining |
| admin-options-preflight-no-key | OPTIONS | `/api/admin/health` | 200 | 200 | OPTIONS preflight is exempt from the API-key gate and returns CORS headers |
| admin-200-with-key | GET | `/api/admin/health` | 200 | 200 | admin endpoint with the correct X-API-Key |
| admin-401-wrong-key | GET | `/api/admin/health` | 401 | 401 | admin endpoint with a wrong X-API-Key |
| admin-401-no-key | GET | `/api/admin/health` | 401 | 401 | admin endpoint with no X-API-Key: empty body plus WWW-Authenticate: X-API-Key |

## Observed formatting facts

```json
{
 "health_timestamp_example": "2026-10-15T07:53:56.552639",
 "health_headers_sample": {
  "Content-Type": "application/json",
  "Transfer-Encoding": "chunked",
  "Date": "Thu, 15 Oct 2026 02:23:56 GMT"
 }
}
```

SSE endpoints (capture separately, they do not terminate):

* POST /api/signals/generate-all/stream
  * GET  /api/candidate-scans/{runId}/stream
  * POST /api/analysis/run-full

Capture the first N events with `curl -N` and store the raw text as the golden.
