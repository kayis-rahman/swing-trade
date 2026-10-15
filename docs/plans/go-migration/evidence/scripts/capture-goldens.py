#!/usr/bin/env python3
"""Capture golden request/response pairs from a running Swing Trade API (Java).

Phase 0 step 4: "Record golden request and response pairs per endpoint: success,
validation error, not found, conflict, rate-limit, unauthorized admin; include
headers and the five SSE event streams."

This script is the reproducible capture half of that deliverable. It is read-only
against the API: every case is a GET unless it is explicitly listed as a writable
probe (validation/conflict probes that fail fast and change nothing).

Usage
-----
    python3 capture-goldens.py --base http://127.0.0.1:8080 --out <dir> [--api-key K] [--max-body 4000]

Outputs
-------
    <dir>/cases/<slug>.json   one file per case: method, path, status, response
                              headers (captured verbatim), and the body (JSON
                              pretty-printed when parseable, else raw, truncated
                              at --max-body chars)
    <dir>/openapi3.json       the live springdoc export
    <dir>/index.md            human-readable index with the observed formatting
                              rules (time/decimal/enum/null shapes)

Exit code is 1 when any case does not return the expected status in the table
below, so the corpus can be re-validated in CI after the Go port exists.

Preconditions
-------------
The Java API must be running against an isolated throwaway database with
scheduling disabled (see phase-0-characterization.md step 1). The capture is
deterministic only on an empty database: every list endpoint returns the empty
collection shape rather than data-dependent rows.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request

# (slug, method, path, expected_status, body_or_None, note)
# Expected statuses are the ones observed on an empty throwaway database;
# a mismatch is a regression signal, not a failure of this script.
CASES = [
    # --- health / platform -------------------------------------------------
    ("health", "GET", "/api/health", 200, None, "HealthController aggregate health"),
    ("health-database", "GET", "/api/health/database", 200, None, "DB health probe"),
    ("health-details", "GET", "/api/health/details", 200, None, "health details"),
    ("health-full", "GET", "/api/health/full", 200, None, "full health (nginx rewrites this in stage)"),
    ("health-llm", "GET", "/api/health/llm", 200, None, "llm health"),
    ("health-market-data", "GET", "/api/health/market-data", 200, None, "market data provider health"),
    ("actuator-health", "GET", "/actuator/health", 200, None, "actuator health, unauthenticated"),
    ("actuator-prometheus", "GET", "/actuator/prometheus", 200, None, "Prometheus scrape endpoint"),
    ("admin-health", "GET", "/api/admin/health", 200, None, "admin health aggregate"),
    ("admin-data-status", "GET", "/api/admin/data/status?symbol=RELIANCE&days=30", 500, None, "admin ingestion status (symbol and days are required)"),
    ("admin-symbols-status", "GET", "/api/admin/symbols/status", 200, None, "symbol universe status"),
    ("admin-kill-switch-status", "GET", "/api/admin/kill-switch/status", 200, None, "kill switch state read"),

    # --- read endpoints: empty-collection and envelope shapes --------------
    ("settings", "GET", "/api/settings", 200, None, "settings bundle"),
    ("settings-llm", "GET", "/api/settings/llm", 200, None, "llm settings"),
    ("settings-trading", "GET", "/api/settings/trading", 200, None, "trading settings"),
    ("settings-scanning", "GET", "/api/settings/scanning", 200, None, "scanning settings"),
    ("settings-discord", "GET", "/api/settings/discord", 200, None, "discord settings"),
    ("settings-gpuhub", "GET", "/api/settings/gpuhub", 200, None, "gpuhub settings"),
    ("settings-pi-status", "GET", "/api/settings/pi/status", 200, None, "pi ssh status"),
    ("signals-latest", "GET", "/api/signals/latest", 200, None, "latest signals list"),
    ("signals-high-confidence", "GET", "/api/signals/high-confidence", 200, None, "high confidence signals"),
    ("signals-date-range", "GET", "/api/signals/date-range?startDate=2026-01-01&endDate=2026-01-31", 200, None, "date-range query params"),
    ("signals-gate-effectiveness", "GET", "/api/signals/gate-effectiveness?from=2026-01-01&to=2026-01-31", 200, None, "gate effectiveness report (from/to required)"),
    ("signals-symbol", "GET", "/api/signals/symbol/RELIANCE", 200, None, "per-symbol signals, empty symbol"),
    ("signals-combined", "GET", "/api/signals/combined/RELIANCE", 200, None,
     "combined signal view: does not answer within 30s on an empty DB with no LLM, so no golden is possible yet (see evidence doc, finding 2)"),
    ("signals-analysis", "GET", "/api/signals/analysis/RELIANCE", 200, None, "stored analysis for a symbol"),
    ("positions", "GET", "/api/positions", 200, None, "open positions list"),
    ("positions-closed", "GET", "/api/positions/closed", 200, None, "closed positions list"),
    ("positions-performance", "GET", "/api/positions/performance", 200, None, "performance summary"),
    ("positions-stats", "GET", "/api/positions/stats", 200, None, "position statistics"),
    ("positions-sector-allocation", "GET", "/api/positions/sector-allocation", 200, None, "sector allocation"),
    ("positions-risk-summary", "GET", "/api/positions/risk-summary", 200, None, "risk summary"),
    ("positions-symbol", "GET", "/api/positions/symbol/RELIANCE", 200, None, "positions for a symbol"),
    ("watchlist", "GET", "/api/watchlist", 200, None, "watchlist list"),
    ("watchlist-all", "GET", "/api/watchlist/all", 200, None, "full watchlist read"),
    ("strategy-types", "GET", "/api/strategy-types", 200, None, "strategy type registry"),
    ("strategy-configs", "GET", "/api/strategy-configs", 200, None, "strategy configuration list"),
    ("strategy-report", "GET", "/api/strategy-report", 200, None, "strategy report"),
    ("backtest-reports", "GET", "/api/backtest/reports", 200, None, "backtest report index"),
    ("backtest-strategies", "GET", "/api/backtest/strategies", 200, None, "backtest strategies"),
    ("sentiment-accuracy", "GET", "/api/sentiment/accuracy", 200, None, "sentiment accuracy"),
    ("sentiment-accuracy-summary", "GET", "/api/sentiment/accuracy/summary", 200, None, "sentiment accuracy summary"),
    ("sentiment-symbol-latest", "GET", "/api/sentiment/RELIANCE/latest", 200, None, "latest sentiment for a symbol"),
    ("sentiment-symbol-history", "GET", "/api/sentiment/RELIANCE/history", 200, None, "sentiment history"),
    ("sentiment-evaluate-status", "GET", "/api/sentiment/evaluate/status", 200, None, "evaluation job status"),
    ("synthesis-evaluations-summary", "GET", "/api/synthesis/evaluations/summary", 200, None, "synthesis evaluation summary"),
    ("holidays", "GET", "/api/holidays", 200, None, "NSE holiday list"),
    ("holidays-today", "GET", "/api/holidays/today", 200, None, "today's holiday state"),
    ("data-status", "GET", "/api/data/status", 200, None, "ingestion status"),
    ("data-export", "GET", "/api/data/export?symbol=RELIANCE&fromDate=2026-01-01&toDate=2026-01-31", 200, None, "data export (CSV)"),
    ("data-pull-progress", "GET", "/api/data/pull/progress", 200, None, "pull progress"),
    ("ingestion-status", "GET", "/api/ingestion/status", 200, None, "ingestion status"),
    ("ingestion-reconcile-out-of-range", "GET", "/api/ingestion/reconcile?from=2026-01-01&to=2026-01-31", 400, None, "reconcile with a 31-day window: range guard"),
    ("ingestion-reconcile", "GET", "/api/ingestion/reconcile?from=2026-01-05&to=2026-01-10", 200, None, "reconcile report inside the allowed window"),
    ("job-runs", "GET", "/api/job/runs", 200, None, "job run list"),
    ("candidate-scans", "GET", "/api/candidate-scans", 200, None, "candidate scan list"),
    ("candidate-scan-settings", "GET", "/api/candidate-scans/settings", 200, None, "candidate scan settings"),
    ("paper-portfolios", "GET", "/api/paper-portfolios", 200, None, "paper portfolio list"),
    ("signal-selections", "GET", "/api/signal-selections", 200, None, "operator signal selections"),
    ("strategy-config-variant", "GET", "/api/strategy-configs/unknown-variant", 400, None, "single config read, unknown id: 400 not 404"),
    ("symbol-strategy-matrix", "GET", "/api/symbols/RELIANCE/strategy-matrix", 200, None, "per-symbol strategy matrix"),
    ("fyers-status", "GET", "/api/fyers/status", 200, None, "fyers auth status"),
    ("news-symbol-latest", "GET", "/api/news/RELIANCE/latest", 200, None, "news for a symbol"),
    ("pdf-symbol-latest", "GET", "/api/pdf/RELIANCE/latest", 200, None, "extracted PDF snapshot"),

    # --- error envelopes ---------------------------------------------------
    ("not-found-signals-symbol", "GET", "/api/signals/symbol/UNKNOWN", 200, None,
     "per-symbol read for a symbol with no rows (empty list, not 404)"),
    ("not-found-unknown-collection", "GET", "/api/definitely-not-a-real-path", 404, None,
     "unmapped path: Spring's default error body"),
    ("validation-watchlist-empty-body", "POST", "/api/watchlist", 400, {},
     "POST with an empty JSON object: bean-validation errors"),
    ("conflict-watchlist-missing-symbol", "POST", "/api/watchlist", 400, {"foo": "bar"},
     "POST with a body that omits the required symbol"),
    ("validation-signals-generate-empty", "POST", "/api/signals/generate", 400, {},
     "POST generate with an empty body"),
    ("reserved-segment-signals-delete-generate", "DELETE", "/api/signals/generate", 200, None,
     "DELETE on 'generate': answered by the {symbol} handler, so 'generate' is NOT a reserved segment"),
    ("reserved-segment-signals-delete-latest", "DELETE", "/api/signals/latest", 400, None,
     "DELETE on the reserved segment 'latest': bare 400 {\"error\": ...} body, not the ApiResponse envelope"),

    # --- intraday paper sessions (POST /api/intraday/paper/sessions) --------
    # This controller is the one that cannot be instantiated by Boot 4's
    # Jackson 3 auto-configuration (evidence doc §1), so its two routes only
    # became capturable after the Jackson 3 fix.
    ("intraday-paper-session-not-found", "GET", "/api/intraday/paper/sessions/gnhf-unknown-run", 404, None,
     "unknown runId: Spring's default 404 error body"),
    ("intraday-paper-session-empty-body", "POST", "/api/intraday/paper/sessions", 400, {},
     "POST with an empty JSON object: bean-validation errors, fieldErrors still serialized as []"),
    ("intraday-paper-session-no-bars", "POST", "/api/intraday/paper/sessions", 200,
     {"symbol": "TCS", "sessionDate": "2026-10-09", "strategyType": "INTRADAY_ORB"},
     "paper replay with no stored bars: 200 with status INCOMPLETE_DATA and no session row persisted"),
    ("intraday-paper-session-non-intraday-strategy", "POST", "/api/intraday/paper/sessions", 400,
     {"symbol": "TCS", "sessionDate": "2026-10-09", "strategyType": "BREAKOUT"},
     "non-INTRADAY_ strategy: empty-bodied 400, no bars loaded and no execution"),
]

# Security cases only make sense against an instance started with
# --app.security.api-key.enabled=true --app.security.api-key=<key>; they are appended
# when --security is passed. Expected statuses assume that instance.
SECURITY_CASES = [
    ("admin-401-no-key", "GET", "/api/admin/health", 401, None,
     "admin endpoint with no X-API-Key: empty body plus WWW-Authenticate: X-API-Key", None),
    ("admin-401-wrong-key", "GET", "/api/admin/health", 401, None,
     "admin endpoint with a wrong X-API-Key", "gnhf-wrong-key"),
    ("admin-200-with-key", "GET", "/api/admin/health", 200, None,
     "admin endpoint with the correct X-API-Key", "provided"),
    ("admin-options-preflight-no-key", "OPTIONS", "/api/admin/health", 200, None,
     "OPTIONS preflight is exempt from the API-key gate and returns CORS headers", None),
    ("rate-limit-headers-on-success", "GET", "/api/signals/latest", 200, None,
     "successful response still carries X-RateLimit-Limit/Remaining", "provided"),
]

SSE_NOTE = """SSE endpoints (capture separately, they do not terminate):

* POST /api/signals/generate-all/stream
* GET  /api/candidate-scans/{runId}/stream
* POST /api/analysis/run-full

Capture the first N events with `curl -N` and store the raw text as the golden.
"""


def fetch(base: str, method: str, path: str, body, api_key: str | None):
    url = base.rstrip("/") + path
    data = None
    headers = {"Accept": "*/*", "User-Agent": "gnhf-golden-capture/1"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if api_key:
        headers["X-API-Key"] = api_key
    if method == "OPTIONS":
        # Required for the CORS preflight to be answered with CORS headers.
        headers["Origin"] = "http://localhost:3000"
        headers["Access-Control-Request-Method"] = "GET"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8", "replace")
            return resp.status, dict(resp.headers), raw
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        return e.code, dict(e.headers), raw
    except Exception as e:
        # Transport-level failure (timeout, reset, DNS): keep the corpus honest by
        # recording status 0 with the reason instead of aborting the whole capture.
        return 0, {}, json.dumps({"capture_error": f"{type(e).__name__}: {e}"})


def parse_body(raw: str):
    if not raw.strip():
        return {"kind": "empty", "value": None}
    try:
        return {"kind": "json", "value": json.loads(raw)}
    except json.JSONDecodeError:
        return {"kind": "text", "value": raw}


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:8080")
    ap.add_argument("--out", required=True)
    ap.add_argument("--api-key", default=os.environ.get("API_KEY"))
    ap.add_argument("--max-body", type=int, default=1200)
    ap.add_argument("--tag", default=os.environ.get("GOLDEN_TAG", "local-throwaway"))
    ap.add_argument("--security", action="store_true",
                    help="also capture the admin API-key and preflight cases")
    ap.add_argument("--only", nargs="+", default=None, metavar="SLUG",
                    help="capture only the cases whose slug contains one of these "
                         "substrings (the public rate limiter is one bucket per "
                         "minute, so a full capture can exhaust it)")
    ap.add_argument("--rate-limit-probe", type=int, default=0,
                    help="fire N requests at --probe-path and record the first rejected response")
    args = ap.parse_args()

    cases_dir = os.path.join(args.out, "cases")
    os.makedirs(cases_dir, exist_ok=True)

    status, headers, raw = fetch(args.base, "GET", "/api-docs", None, args.api_key)
    if status != 200:
        print(f"springdoc export unavailable at /api-docs: HTTP {status}", file=sys.stderr)
        return 1
    with open(os.path.join(args.out, "openapi3.json"), "w") as fh:
        json.dump(json.loads(raw), fh, indent=1, sort_keys=True)

    if args.security:
        for case in SECURITY_CASES:
            CASES.insert(0, case)

    failures = []
    index_rows = []
    selected = [case for case in CASES if not args.only or any(token in case[0] for token in args.only)]
    for case in selected:
        slug, method, path, expected, body, note = case[:6]
        # 7th element, when present, is the auth directive: None sends no key,
        # "provided" (or absent) uses --api-key, any other string is sent verbatim.
        directive = case[6] if len(case) == 7 else "provided"
        if directive is None:
            key = None
        elif directive == "provided":
            key = args.api_key
        else:
            key = directive
        status, headers, raw = fetch(args.base, method, path, body, key)
        parsed = parse_body(raw)
        record = {
            "slug": slug,
            "note": note,
            "tag": args.tag,
            "request": {"method": method, "path": path, "body": body},
            "response": {
                "status": status,
                "expected_status": expected,
                "headers": {k: v for k, v in sorted(headers.items())},
                "body": parsed,
            },
        }
        truncated = False
        if parsed["kind"] == "json":
            text = json.dumps(parsed["value"], indent=1, sort_keys=True)
        else:
            text = parsed.get("value") or ""
        if len(text) > args.max_body:
            text = text[: args.max_body] + "\n...TRUNCATED"
            truncated = True
        parsed["value"] = text
        record["response"]["body_truncated"] = truncated
        with open(os.path.join(cases_dir, f"{slug}.json"), "w") as fh:
            json.dump(record, fh, indent=1, sort_keys=True)
        if status != expected:
            failures.append((slug, method, path, expected, status))
        index_rows.append((slug, method, path, status, expected, note))
    # The unmistakable formatting facts, pulled straight out of one response.
    fmt = {}
    try:
        h, hh, raw = fetch(args.base, "GET", "/api/health", None, args.api_key)
        fmt["health_timestamp_example"] = json.loads(raw).get("timestamp")
        fmt["health_headers_sample"] = {k: v for k, v in hh.items() if k.lower() in (
            "content-type", "date", "cache-control", "transfer-encoding")}
    except Exception as exc:  # pragma: no cover - diagnostics only
        fmt["error"] = repr(exc)

    with open(os.path.join(args.out, "index.md"), "w") as fh:
        fh.write("# Golden corpus index\n\n")
        fh.write(f"Capture tag: `{args.tag}`. Base: `{args.base}`.\n\n")
        fh.write("| case | method | path | status | expected | note |\n")
        fh.write("|---|---|---|---|---|---|\n")
        for slug, method, path, status, expected, note in index_rows:
            flag = "" if status == expected else " **MISMATCH**"
            fh.write(f"| {slug} | {method} | `{path}` | {status} | {expected}{flag} | {note} |\n")
        fh.write("\n## Observed formatting facts\n\n```json\n")
        fh.write(json.dumps(fmt, indent=1))
        fh.write("\n```\n\n")
        fh.write(SSE_NOTE)

    if args.rate_limit_probe:
        probe_path = "/api/signals/latest"
        rejected = None
        for i in range(args.rate_limit_probe):
            status, headers, raw = fetch(args.base, "GET", probe_path, None, args.api_key)
            if status != 200:
                rejected = {"attempt": i + 1, "status": status,
                            "headers": {k: v for k, v in sorted(headers.items())},
                            "raw_body": raw[: args.max_body], "body_length": len(raw)}
                break
        with open(os.path.join(args.out, "cases", "rate-limit-429.json"), "w") as fh:
            json.dump({"slug": "rate-limit-429", "tag": args.tag,
                       "note": "public fixed-window rate limiter, probed with "
                               f"{args.rate_limit_probe} requests to {probe_path}",
                       "probe": {"path": probe_path, "requests": args.rate_limit_probe},
                       "first_rejection": rejected}, fh, indent=1, sort_keys=True)
        print(f"rate-limit probe: first rejection at attempt "
              f"{rejected['attempt'] if rejected else 'none'} status "
              f"{rejected['status'] if rejected else 'n/a'}")

    print(f"captured {len(index_rows)} cases into {cases_dir}")
    if failures:
        print("status mismatches:", file=sys.stderr)
        for slug, method, path, expected, status in failures:
            print(f"  {slug}: {method} {path} expected {expected} got {status}", file=sys.stderr)
        return 1
    print("all cases matched their expected status")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
