# Codex project direction

Apply the shared repository instructions in [`AGENTS.md`](../AGENTS.md) first. Work as a senior/staff engineer and software architect: trace ownership and data flow, respect module boundaries, favor the smallest maintainable change, and verify behavior rather than relying on inspection alone.

## Skill routing

Use repo skills in `.agents/skills/` for their matching workflows:

- `backend-dev` — Java/Spring backend, persistence, integrations, and backend tests.
- `dashboard-dev` — Vue, TypeScript, Pinia, dashboard behavior, and frontend tests.
- `dev-stack` — local service operation and troubleshooting. Keep local and stage targets distinct.
- `go-platform-dev` — Go workspace, modules, decimals, SQLC, and parity contracts.
- `llm-runtime-settings` — live LLM settings resolution; use as a backend-dev companion.
- `stage-go-cutover` — stage-only Go API replacement artifacts and evidence.
- `crit-review` — only when the user explicitly asks to use Crit.

Invoke a skill by name when helpful; do not force a skill onto unrelated work. The instructions live in each skill’s `SKILL.md`.

## Architecture map

- `backend/` is a Java 21 Spring Boot multi-module Gradle project: `core` owns domain models and ports; `data` owns persistence and market-data clients; `strategy` owns indicators, signals, and backtests; `llm` owns news and sentiment integrations; `broker` owns paper trading and risk controls; `gpuhub` owns the GPUHub client; `api` owns REST and orchestration.
- Dependency direction: `api → strategy, llm, broker, gpuhub, data, core`; `broker → strategy, data, core`; `strategy → data, llm, core`; `llm → data, core`; `data → core`; `gpuhub` and `core` are standalone. Do not add reverse dependencies.
- Keep domain logic out of `api` and `data`; keep persistence and HTTP calls out of `strategy`. Use `BigDecimal` for monetary values.
- `dashboard/` is Vue 3, TypeScript, Vite, Pinia, Vitest, and Playwright. `infra/` contains deployment and local-stack configuration. `docs/` is the canonical project documentation.
- `go/` is a Go workspace preserving Java's domain ownership across `core`, `data`, `strategy`, `llm`, `broker`, `gpuhub`, and `api` modules. Dependency direction mirrors Java at the module level (for example `api → core, data, ...`); do not add reverse imports. Use fixed-scale decimals for money/percent/indicator math.

## Verification

Run `./bin/verify-changes` from the repository root for automatic backend/dashboard checks based on staged, unstaged, and untracked files. Use `--dry-run` to inspect the selected groups first. The helper intentionally uses non-fixing dashboard lint and build commands. For a focused task, run the narrowest relevant check first; expand to module or project suites when a change crosses boundaries or has broader impact.

```bash
# Backend, from backend/
./gradlew :data:test :api:test --no-daemon
./gradlew build

# Dashboard, from dashboard/
yarn typecheck
yarn test:run
yarn build

# Local stack, from the repository root
./dev-stack.sh status
curl -sS http://localhost:8080/actuator/health
```

Integration tests may need Docker and a test database; do not confuse these with the local development database. Report checks actually run and any that could not be run.

## Operational safety

- Never commit credentials, tokens, `.env` files, generated logs, build output, or runtime data. Use `infra/env/.env.example` as the template. The local `.env` is sourced by stack scripts, not auto-loaded by Spring Boot.
- Treat database cleanup, stage deployment, and live-broker actions as destructive. Confirm the exact target and requested scope before acting.
- The local development database is PostgreSQL at `192.168.0.100:5435`, database `swingtrade_db`. Stage is separate, with API on `:8081` and database `swingtrade_stage`; use stage commands only when explicitly working on stage.
- Integration and read-only subagents are configured in `.codex/agents/`. Delegate only bounded, independent work; the parent agent owns integration, edits, and final verification.
