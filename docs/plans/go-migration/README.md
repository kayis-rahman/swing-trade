# Java to Go migration plan

Status: Phase 1 local no-traffic skeleton authorized. Phase 0 and G0 were waived by captain decision on 2026-10-17; later phases still require explicit approval at their gates.
Source analysis: a read-only scout of the repository at `c691c504` (module sizes, endpoints, schedulers, deployment, observability, safety flags); file references point at this repository. Repo baseline: `c691c504`.

## Decisions (captain, 2026-10-13)

| Topic | Decision |
|---|---|
| Motivation / success metric | Pi memory and footprint; Go must beat the native-image Java baseline measured in Phase 0 |
| Dashboard | Vue app unchanged behind nginx |
| Cutover shape | Gated big-bang: phases and evidence gates G0 to G7, one traffic and job-scheduling switch at G8 (Java `app.scheduling.enabled=false`, Go advisory locks) |
| Live broker (Kite) | Interface plus refusing stub; live code behind a `live` build tag, off in every released binary; paper-only is the only enabled mode |
| Migrations | Java keeps Flyway until G8; goose takes over from a no-op V76 baseline at cutover; additive-only schema rules in [rollback.md](rollback.md) |
| Shadow evidence | 1 trading week for reads, 2 weekly cycles for schedulers, zero tolerance on decimal paths |
| Security scope | Parity only (secret rotation tracked separately) |
| Logs | JSON |
| Rollback triggers | See [rollback.md](rollback.md) |
| End state | Retire Java at G8 |
| Structure and process | Same module separation, same scanners, test-first (see [standards.md](standards.md)) |
| G0 and native-image comparison | Captain decision (2026-10-17): waive Phase 0/G0 and the native-image baseline; stop evidence collection, write no G0 report, and proceed with Phase 1 locally under [the authorization record](phase-1-authorization.md). |

## Document map

| File | Purpose |
|---|---|
| [standards.md](standards.md) | Module layout, dependency rules, static scanning, coding standards, TDD loop |
| [phase-0-characterization.md](phase-0-characterization.md) | Authorized now: baseline, goldens, ledger, traceability |
| [phase-1-platform-skeleton.md](phase-1-platform-skeleton.md) | Go workspace, tooling, observability, no traffic |
| [phase-1-authorization.md](phase-1-authorization.md) | Phase 0/G0 waiver and Phase 1 operational boundaries |
| [phase-1-config.md](phase-1-config.md) | Go environment mapping, safe local commands, database-test limits |
| [phase-1-metrics.md](phase-1-metrics.md) | Micrometer-compatible platform metric names for the skeleton |
| [phase-2-read-edge.md](phase-2-read-edge.md) | Read-only endpoints, shadow comparison |
| [phase-3-data-ingestion.md](phase-3-data-ingestion.md) | Providers, calendar, ingestion jobs |
| [phase-4-strategy-backtest.md](phase-4-strategy-backtest.md) | Indicators, strategies, backtests |
| [phase-5-llm-news-sentiment.md](phase-5-llm-news-sentiment.md) | LLM, news, sentiment, synthesis |
| [phase-6-orchestration-writes.md](phase-6-orchestration-writes.md) | Orchestrator, signals, SSE, write APIs, schedulers |
| [phase-7-paper-trading-risk.md](phase-7-paper-trading-risk.md) | Paper engine, risk controls, kill switch |
| [phase-8-cutover-retirement.md](phase-8-cutover-retirement.md) | Switch, rollback window, Java retirement |
| [risks.md](risks.md) | Risk register |
| [rollback.md](rollback.md) | Triggers and procedures |
| [acceptance.md](acceptance.md) | Final acceptance criteria |
| [feature-ledger.md](feature-ledger.md) | Per-feature tracking template and initial rows |

## Gate protocol
A gate is passed only when: all exit criteria in the phase file are met with evidence attached to the ledger; the captain approves in writing; `docs/status.md` is updated after verification. No phase starts early. Findings that change a decision are brought back to the captain, not absorbed silently.
