# Gate readiness summaries (Go port, paperwork only)

Prepared 2026-11-02 on branch `fm/swing-trade-go-gate-evidence-v1`. Docs only.
No code changed. No gate is claimed to pass without evidence. Captain signature line at the
end of each gate.

Current baseline: merged PRs through Phase 7 plus Phase 6 slice F (PR #165; Discord notification
services and schedulers ported; Telegram intentionally not ported) and the signal
pipeline/scans slice (#166; candidate scan manager and signal pipeline adapter).
Phase 8 (cutover and Java retirement) remains out of scope and requires written captain approval.

## Gate G4 - Strategy, indicators, backtest engine

Reference: `phase-4-strategy-backtest.md` -> `## Gate G4 exit criteria`.

| Exit criterion | Status | Evidence in repo | Missing |
| --- | --- | --- | --- |
| Zero diff on decimal paths | OPEN | `go/strategy/decimal_num.go`, `decimal_math.go`, `market_context.go`, `market_context_indicators.go`; `F-STRAT-IND-01..11` rows are `parity-pending` | Golden-signal comparison over the full watchlist history; parallel-run output; Java golden vectors beyond the committed red tests |
| Trade lists identical | OPEN | Intraday backtest engine now ported (`go/strategy/intraday_backtest_engine.go`); backtest/portfolio engines still absent | Backtest engines and their translated Java tests; trade-list comparison report |
| Documented tolerances only on float paths that are float in Java too | PARTIAL | Porting notes in `phase-4-strategy-backtest.md`; float-path rationale documented | Final tolerance table with measured diffs; floats remain parity-pending |
| Performance recorded | OPEN | none | Performance comparison on the Pi against Java |

Ledger: no `F-STRAT-IND-*` row moved to `closed`; all remain `parity-pending`.

Gate G4 verdict: NOT READY. Signature: ______________ Date: ______________

## Gate G5 - LLM, news, sentiment, synthesis

Reference: `phase-5-llm-news-sentiment.md` -> `## Gate G5 exit criteria`.

| Exit criterion | Status | Evidence in repo | Missing |
| --- | --- | --- | --- |
| All recorded-response suites green | PARTIAL | `go/llm/client_test.go`, `sentiment_analyzer_test.go`, `synthesis_output_test.go`, `validator_test.go`, `pdf_extraction_test.go` and others | Full provider recorded-response corpus (all backends/sources) and green recorded suite across modules |
| Degraded and fallback behavior match | PARTIAL | `news_circuit_breaker.go`, fallback sentiment path in `sentiment_analyzer.go` | Drill evidence for provider-down, malformed JSON, timeouts, SSH failure; stage_degraded parity report |
| Metrics match | OPEN | Meter usage present; full metrics inventory still tracks Go parity | Metric golden comparison for LLM/sentiment counters |
| No secret in fixtures | OPEN | not audited in this paperwork pass | Secret scan report over recorded fixtures |

Ledger: Phase 5 rows remain as currently marked; no row moved to `closed`.

Gate G5 verdict: NOT READY. Signature: ______________ Date: ______________

## Gate G6 - Orchestration, signals, SSE, write APIs

Reference: `phase-6-orchestration-writes.md` -> `## Gate G6 exit criteria`.

| Exit criterion | Status | Evidence in repo | Missing |
| --- | --- | --- | --- |
| Run outputs equal | PARTIAL | `go` job orchestrator package, signal arbiter, candidate scans and writes | Full E2E comparison on a scratch database; Java-versus-Go run summaries and persisted signals |
| SSE streams equal | OPEN | `WriteSSE` helper exists in `go/api/sse.go` and is exercised by `go/api/helpers_test.go`, but no stream endpoint or consumer uses it | SSE stream endpoints/consumers, heartbeat and ordering implementation, and parity tests |
| Playwright green | OPEN | existing Playwright suite only | Unchanged Playwright run against Go on scratch stack |
| Two weekly cycles without duplicate or missed jobs | OPEN | scheduler-off gates remain default false | Two-cycle scheduler soak report (digests included) |

In-flight at baseline: none; the signal pipeline/scans slices landed in #166. Telegram is intentionally not ported (Discord schedulers landed in slice F).

Ledger: no Phase 6 row moved to `closed`; several touched rows remain `parity-pending` or `not-started`.

Gate G6 verdict: NOT READY. Signature: ______________ Date: ______________

## Gate G7 - Paper trading, risk, kill switch

Reference: `phase-7-paper-trading-risk.md` -> `## Gate G7 exit criteria`.

| Exit criterion | Status | Evidence in repo | Missing |
| --- | --- | --- | --- |
| Zero divergence across the shadow window including drills | OPEN | `go/broker/paper_engine.go`, `position.go`, risk controls, `kill_switch.go` | Shadow run with Go computing decisions while Java remains writer; orders/fills/costs/positions/P&L equality |
| Kill switch honored within 1 s | PARTIAL | `kill_switch.go` | Timed drill evidence under 1 s |
| Live-path absence verified | PARTIAL | `go/broker/live_broker.go` behind `//go:build live`, refusing stub | CI build/symbol check evidence against the released binary |

Related acceptance fixture replay (25 bars, 2 orders, 1 trade, costs 47.57, net -27.6802601): OPEN, Java intraday acceptance report exists; no Go-side replay/parity evidence found.

Ledger: Phase 7 rows (broker paper engine, risk) remain `not-started`/`parity-pending` as marked.

Gate G7 verdict: NOT READY. Signature: ______________ Date: ______________

## Notes

- Golden criterion rechecks in progress in other slices are not folded into these verdicts.
- Phase 8 (cutover and Java retirement) is out of scope here and needs the captain's written approval.
