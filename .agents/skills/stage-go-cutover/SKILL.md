---
name: stage-go-cutover
description: Implement or inspect stage-only Go API cutover work. Use for infra/docker-compose.go-stage.yml, infra/Dockerfile.go-stage, nginx shadow compose, stage deploy scripts, or stage deployment evidence.
---

# Stage Go cutover

Work as a senior/staff engineer: treat stage replacement as API-only and reversible until G8.

## Scope

In scope: no-traffic Go API replacement on stage; external stage network/aliases; CPU/memory caps; rollback container preservation; evidence requirements.

Out of scope (do not generalise): local development stack operations (`dev-stack`), live broker actions, and production deployment.

## Invariants

- Go stage service is API-only: the stage environment's `DATABASE_URL` GitHub secret supplies the stage-only database connection for reads; keep background ingestion disabled until G8.
- Keep the stopped Java API container for rollback. If it is unavailable, use a previous Go container only with explicit authorization.
- Preserve external network naming and the `8081:8080` mapping.
- Keep container cap at 1.5 CPU and 1 GB memory (compose lines 25-32).
- Nginx shadow mirror remains disabled by default and covers only explicitly selected GET paths.
- Destructive stage commands require exact target confirmation and authorization.

## Verification

- Inspect compose and evidence before changing stage runtime state.
- Report checks actually run and any that could not be run.

Preserve unrelated worktree changes.
