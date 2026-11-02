# Java test traceability matrix (Go port)

Status as of 2026-11-02: Phase 0 step 13 was still open. Method-level mapping for Java
tests has not been completed repository-wide. This file records that honest state so
unmapped tests are visible to the gate checks.

## Current state

- Expanded ledger rows name Java test classes for endpoints, jobs, metrics, security,
  entities, and repositories.
- Indicator (§12) and domain (§13) rows name Java test classes/methods.
- Method-level `Java test method -> Go test` mapping: INCOMPLETE. Many rows still list
  classes only; no row carries commit-level red/green mapping for all Java test methods.
- Unmapped Java tests currently: all Java test methods not explicitly listed in the ledger
  rows above remain unmapped. This blocks acceptance criterion 7 and all of G4-G7 gates
  that require "every Java test mapped or dropped with a reason".

## Gate-blocking unmapped areas

| Area | Java test surface | Mapping status | Missing |
|---|---|---|---|
| Core domain model and ports | 81 types in `backend/core/src/main/java/com/swingtrade/domain` | not started | F-DATA-ENTITIES expansion (§11) |
| Indicators, strategies, gates | `backend/strategy` test tree | partial rows (F-STRAT-IND-01..21), method mapping partial | method-level matrix and golden vectors |
| Backtest engines, walk-forward, DSR, cost models | `backend/strategy` backtest tests | not expanded | §11 F-STRAT-BT |
| LLM backends, news sources, prompts, risk controls | `backend/llm`, `backend/broker`, `backend/data/client` | not expanded | §11 rows |
| Scheduling settings | application properties plus scheduler tests | partial | property rows |
| Broker/paper trading (13.6k Java test lines) | `backend/broker/src/test` | only summary rows | full method mapping |

## Conclusion

Do not claim any gate G4-G7 passes on test traceability until this matrix is populated.
