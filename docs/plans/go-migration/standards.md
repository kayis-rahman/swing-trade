# Standards: structure, scanning, TDD

## Module layout (mirrors Gradle)
One Go workspace (`go.work`), one Go module per Gradle module, so the compiler rejects reverse dependencies.

| Java | Go module | May import |
|---|---|---|
| core | swingtrade/core | stdlib |
| data | swingtrade/data | core |
| strategy | swingtrade/strategy | core, data, llm |
| llm | swingtrade/llm | core, data |
| broker | swingtrade/broker | core, data, strategy |
| gpuhub | swingtrade/gpuhub | none |
| api | swingtrade/api, cmd/swingtrade-api | all |

Package layers inside a module keep the Java names and roles (api: controller, service, dto, config, scheduler, metrics; data: entity, repository, store, client, service; broker: risk, manager, scheduler, service). Ports and interfaces live in core and are implemented in data and broker. Each Java class maps to a named Go file or type recorded in the feature ledger.

## Static scanning (each Java gate gets a Go equivalent)

| Java | Go |
|---|---|
| Checkstyle | gofmt, goimports, revive |
| PMD | staticcheck, go vet, errcheck, gocritic, gosec, ineffassign, unused, bodyclose, sqlclosecheck, nilerr |
| ArchUnit | separate modules, depguard, go-arch-lint, arch test |
| JaCoCo 80% | per-package 80% coverage gate from the first commit |
| new | go test -race, govulncheck, forbidigo (no float64 in money packages, no time.Now outside the clock port), fuzz targets for parsers |
| bin/verify-changes | extended to select Go checks from changed paths |

Translate the Checkstyle XML and PMD ruleset rule by rule into a mapping table in Phase 1; unmapped rules are listed with a reason.

## Coding standards
- Money and P&L use one decimal package with explicit scale and rounding constants mirroring `FinancialScale` and `BacktestConfig`; never float64.
- `context.Context` first argument; no package globals; errors wrapped with `%w`; no panics outside main.
- Time: IST wall-clock with a fixed `Asia/Kolkata` location; serialization identical to Java output (captured goldens decide).
- SQL via pgx and sqlc, explicit transactions in the service layer, optimistic locking as explicit version checks with the existing retry policy.
- Logs via `log/slog` JSON with `traceId`, `symbol`, `module` fields. Metrics via client_golang with Micrometer-compatible names.
- Secrets only from env or files. Never copy the committed Finnhub key into any Go file or fixture.

## TDD loop (every feature)
1. Ledger row exists (Java classes, endpoints, jobs, tables, metrics, settings, Java tests).
2. Read the Java implementation and tests; capture goldens from the running Java system for what tests do not pin.
3. Red: write Go tests translated from the Java tests plus golden cases; commit them failing before any implementation.
4. Green: implement the minimum to pass.
5. Refactor within boundaries and lint rules.
6. Parity gate: golden or shadow comparison; attach evidence.
7. Close the ledger row. Every Java test method maps to a Go test or is dropped with a recorded reason (traceability matrix); unmapped tests block the gate.
