# Fixture secret scan report

Generated: 2026-11-05 (UTC session)
Audited commit: 1374d05745789f7a2b091d6638892f6c0388ab8a (branch `fm/swing-trade-go-port-completion-v2`)
Command: `./bin/scan-recorded-fixtures.sh`
Scan scope: tracked `*.json`, `*.yaml`, `*.yml`, `*.txt` files anywhere in the repository
Patterns: AWS access keys (`AKIA...`), GitHub personal access tokens (`ghp_...`), OpenAI-style keys (`sk-...`), and `api[_-]?key` assignments with a value of 8+ characters.

Result: PASS - exit status 0.

Recorded output:
```
Scanning tracked test fixtures for obvious secret/token patterns...
No obvious secrets found in tracked fixtures.
```

This report is supporting evidence only for the G5 "No secret in fixtures" exit criterion. It does
not, by itself, mark G5 ready; gate verdicts remain the captain's decision per
`gate-readiness.md`.
