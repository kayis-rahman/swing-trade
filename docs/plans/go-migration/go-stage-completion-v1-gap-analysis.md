# Go stage completion v1 - gap analysis

See durable task evidence at
`/Users/kayisrahman/Documents/workspace/firstmate/data/swing-trade-go-stage-completion-v1/gap-analysis.md`
for the full analysis. This repository copy stores the executable summary used
by the completion PR.

## Not ported / not proven

- Feature ledger: 8 closed, 76 parity-pending, 320 not-started (Phases 4-7).
- Gates G4-G7 each NOT READY per `gate-readiness.md`.

## Stage deployment

- Automation exists (`go-stage-deploy.yml`, `deploy-go-stage.sh`, compose, verifiers).
- Secrets consumed from the piworm operator env file; never committed. No stage run
  has been executed by this task. The authoritative verifier gate remains required.

## Salvaged branch assessment

- `fm/swing-trade-go-stage-deploy-gh-workflows-macmini-v1`: uncommitted edits to go-ci
  and integration test build tags are not adopted; assessed risky and left for review.
- `fm/swing-trade-go-stage-playwright-parity-v1`: 15 commits (stage DB wiring, settings
  stage-safe, health assertion). Recommended for selective cherry-pick in follow-up.

## 14-day rollback window (stage -> Java)

- Keep stopped Java API container (`swing-trade-stage-api` renamed rollback) for 14 days.
- Additive schema only; Java Flyway validation retained.
- Re-enable Java scheduling, route nginx back to Java, reconcile data; restore backup
  if irreconcilable. Document incidents in rollback log.
