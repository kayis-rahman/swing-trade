# Risk register

| ID | Risk | Mitigation | Phase |
|---|---|---|---|
| R1 | Numeric parity of TA4j DecimalNum behavior | Java-generated golden vectors, zero tolerance, fuzzing | 4 |
| R2 | Paper and risk safety re-implementation | Last phase, shadow, drills, no live path in released binaries | 7 |
| R3 | Hidden Spring/JPA behavior (transactions, @Version, lazy loads) | Ledger rows per entity and service, explicit transaction tests | 2 to 7 |
| R4 | Single-instance schedulers without locks | Global `app.scheduling.enabled=false` at cutover, per-job advisory locks, scratch databases before G8 | 3, 6 |
| R5 | Flyway ownership | Java keeps Flyway until G8; goose V76 no-op baseline, additive-only schema rules (rollback.md) | 8 |
| R6 | JSON and time contract drift | Goldens, comparator, Playwright suite | 0, 2 |
| R7 | LLM stack specifics | Recorded responses, behavior tests | 5 |
| R8 | News scraping fragility | Recorded fixtures, parser fuzzing | 5 |
| R9 | SSE semantics | Event stream goldens | 6 |
| R10 | Prometheus series continuity | Naming table, alias endpoints, Grafana and alert checks | 1 |
| R11 | No active CI | Local verify script parity, optional runner decision | 1 |
| R12 | Features keep landing on Java during a long port | Feature freeze rules per phase; any new Java feature gets a ledger row | all |
| R13 | Committed secret | Rotate separately, never copy | 0 |
| R14 | Big-bang cutover removes incremental production exposure | Long shadow windows, weekly cycles, rollback drill, warm Java | 2 to 8 |
