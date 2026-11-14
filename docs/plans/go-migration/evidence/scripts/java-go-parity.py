#!/usr/bin/env python3
"""Side-by-side Java/Go endpoint parity probe.

Sends the same request to a running Java API and a running Go API that share one
seeded throwaway database, then compares status codes and JSON bodies after
removing fields that legitimately differ per process (clocks, uptime, runtime
version, memory). See java-go-endpoint-parity.md for the run that produced the
recorded report.

Usage:
    python3 java-go-parity.py --java http://127.0.0.1:8080 --go http://127.0.0.1:8090 \
        --cases cases.json --out report.json

cases.json is a list of {"id", "method", "path", "body"?, "unordered"?} objects.
"unordered" names object keys whose list values Java leaves in no defined order
(a HashSet, or a repository findAll() without ORDER BY); those lists are compared
as multisets. Requests run in order, Java first then Go, so a write case can be
followed by a read case that proves both implementations persisted the same state.
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request

VOLATILE_KEYS = {"timestamp", "uptime", "uptimeSeconds", "javaVersion", "memory",
                 "freeMemory", "totalMemory", "maxMemory", "usedMemory", "responseTimeMs",
                 "checkedAt", "lastChecked", "traceId", "path"}


def call(base: str, case: dict) -> tuple[int, object]:
    data = None
    headers = {"Accept": "application/json"}
    if case.get("body") is not None:
        data = json.dumps(case["body"]).encode()
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(base + case["path"], data=data, method=case["method"], headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            status, raw = response.status, response.read()
    except urllib.error.HTTPError as error:
        status, raw = error.code, error.read()
    try:
        return status, json.loads(raw) if raw else None
    except json.JSONDecodeError:
        return status, raw.decode(errors="replace")


def normalize(value: object, unordered: frozenset[str] = frozenset()) -> object:
    if isinstance(value, dict):
        out = {}
        for key, item in sorted(value.items()):
            if key in VOLATILE_KEYS:
                continue
            if key == "systemInfo" and isinstance(item, dict):
                # Health systemInfo reports the serving runtime (JVM or Go) under
                # Java's keys by design, so only the key set is comparable.
                out[key] = sorted(item)
                continue
            item = normalize(item, unordered)
            if key in unordered and isinstance(item, list):
                item = sorted(item, key=lambda v: json.dumps(v, sort_keys=True))
            out[key] = item
        return out
    if isinstance(value, list):
        return [normalize(v, unordered) for v in value]
    return value


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--java", required=True)
    parser.add_argument("--go", required=True)
    parser.add_argument("--cases", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    with open(args.cases, encoding="utf-8") as handle:
        cases = json.load(handle)
    results = []
    for case in cases:
        java_status, java_body = call(args.java, case)
        go_status, go_body = call(args.go, case)
        unordered = frozenset(case.get("unordered", []))
        match = java_status == go_status and normalize(java_body, unordered) == normalize(go_body, unordered)
        results.append({"id": case["id"], "method": case["method"], "path": case["path"],
                        "javaStatus": java_status, "goStatus": go_status, "match": match,
                        "java": java_body, "go": go_body})
        print(f"{'MATCH' if match else 'DIFF '} {java_status}/{go_status} {case['method']} {case['path']} ({case['id']})")
    with open(args.out, "w", encoding="utf-8") as handle:
        json.dump(results, handle, indent=2, sort_keys=True)
    return 0 if all(r["match"] for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
