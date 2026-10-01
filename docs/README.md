# Documentation guide

`README.md` is the short project overview. `AGENTS.md` and `CLAUDE.md` describe how Codex and Claude Code work in the repository.

## Canonical documentation

- `docs/status.md` — verified project state and remaining pilot risks
- `docs/infra/` — local, stage, and monitoring operations
- `docs/infra/local-e2e-gate.md` — the local end-to-end quality gate: layers, hermeticity, and what it cannot prove
- `docs/infra/stage-validation-runbook.md` — read-only checks against the deployed stage stack
- `docs/api-references/` — external API behavior and limitations
- `docs/backtesting.md` — backtest rules and assumptions
- `docs/issues/` — scoped product/engineering work items
- `docs/specs/` — approved design documents
- `docs/plans/` — active implementation plans
- `docs/plans/archive/` — historical plans retained for context

Use one canonical file per topic. Do not add plans under `.claude/plans/`; Claude-specific session state belongs under `.claude/` and should not duplicate project documentation.

## Updating docs

Prefer concise, current instructions over copied implementation history. Mark claims as verified only after running the relevant command or smoke test. When behavior changes, update the closest canonical document and link to it from the README rather than duplicating the explanation.
