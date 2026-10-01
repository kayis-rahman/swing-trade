# PR #94 Review Report — MLX Backend Support

## Status: DEFERRED BY CAPTAIN (2026-10-01)

The captain chose to set this work aside rather than reroute the pipeline review, which is blocked by a third-party documentation host returning 404. The branch and all commits are preserved; the remote branch is untouched; the PR is not closed.

## PR Summary

- **URL**: https://github.com/kayis-rahman/swing-trade/pull/94
- **Title**: feat(llm,api,frontend): add MLX backend support for local Apple Silicon LLM inference
- **Original head**: 8d1ec0ba (branch `worktree-status-md`)
- **Current head**: 3e6e980fa249d22d30d73689ad4ea139a49b0255
- **Base**: origin/main (6ab6a047)
- **Branch**: `worktree-status-md`

## What the PR Contains

Adds MLX (Apple Silicon) as a 7th LLM backend alongside local, pi_ssh, openai, ollama, laya, and pi_agent:

- **MlxServerManager** (new, 525 lines): manages `mlx_lm.server` process lifecycle — start, stop, health check, idle auto-stop, restart on model change
- **LlmBackendSelector**: adds `MLX("mlx")` to the Backend enum
- **LlmConfig**: adds `mlxChatModel` bean using `LlmProperties.Providers.mlx` (type-safe config)
- **LlmProperties**: adds `mlx` Provider with `llm.providers.mlx.base-url` / `llm.providers.mlx.model` properties
- **LlmClientProvider / LlmServerManagerProvider**: wire MLX into routing
- **SettingsController**: MLX settings in getLlmSettings, restart on model change, `/settings/mlx/start|stop|status` and `/settings/test/mlx` endpoints
- **Dashboard**: MLX backend option, model selector, server lifecycle controls, test inference button
- **Tests**: MlxServerManagerTest (202 lines), updated LlmClientProviderRoutingTest, LlmConfigMultiClientTest, SentimentServiceCoverageTest, SettingsControllerDefaultsTest, SettingsView.test.ts, settings.test.ts, api/settings.test.ts
- **Docs**: docs/plans/mlx-backend.md

## Rebase

The original PR was 358 commits behind main. Rebased onto current main (6ab6a047) with 8 conflicting files resolved:
- Kept main's `LlmProperties` architecture, gpuhub endpoints, secret redaction, and gpuhub alias
- Layered MLX on top as a 7th backend using main's type-safe config pattern
- Adapted all dashboard code to main's `apiRequest`/`confirmed()` pattern
- Updated tests for new constructor signatures and bean counts

## Validation

- **Backend**: Full test suite passes (LLM: 277+ tests, API: 441+ tests, all modules green)
- **Frontend**: All 360 Vitest tests pass; typecheck and lint clean
- **no-mistakes pipeline**: 12 fix rounds applied across review findings

## Review Findings and Decisions

### Ask-User Findings (captain decisions via firstmate)

1. **mlx-credential-forwarding** (error): Unauthenticated caller could PUT attacker-controlled `mlx.server.url`, then POST `/settings/test/mlx` which forwarded `openai.api_key` to that URL.
   - **Decision**: OPTION 2 — single-user internal app, no auth to add. MLX Test uses only persisted `mlx.server.url`; don't forward OpenAI key to MLX server. Add tests.
   - **Applied**: Pipeline fixed the test probe to not forward credentials.

2. **mlx-remote-start** (warning): `ensureRunning()` launches MLX on API host even when configured URL points to remote Mac.
   - **Decision**: MLX runs locally on API host only; docs say local-only.
   - **Applied**: Pipeline bound MLX to loopback, updated docs.

3. **mlx-duplicate-lifecycle** (warning): `infra/mlx-llm.sh` duplicates MlxServerManager lifecycle.
   - **Decision**: Remove the standalone script.
   - **Applied**: `infra/mlx-llm.sh` deleted.

4. **mlx-runtime-credential-forwarding** (error): MLX chat model reuses OpenAI credential; public settings endpoint accepts `mlx.server.url`.
   - **Decision**: Same as #1 — prefer not forwarding OpenAI key to MLX.
   - **Applied**: Pipeline configured MLX with no provider key.

5. **mlx-remote-endpoint-exceeds-local-scope** (warning): Remote MLX endpoint mode exceeds local-only scope.
   - **Decision**: Remove remote-endpoint path, keep MLX local.
   - **Applied**: Pipeline made remote stop a no-op, kept local lifecycle.

6. **mlx-endpoint-receives-reused-provider-key** (error): MLX model reuses OpenAI/gpuhub key.
   - **Decision**: Same as #1 — don't forward credentials to MLX.
   - **Applied**: Pipeline configured MLX with no provider key.

### Auto-Fix Findings (all applied by pipeline)

1. **mlx-stop-unrelated-process**: Port-wide kill could terminate unrelated processes → restricted to verified MLX-owned processes
2. **mlx-idle-stop-never-triggers**: `idleCheckTime` never updated → track real activity, respect in-flight requests
3. **mlx-concurrent-start-hang**: Waiting callers loop forever on startup failure → bounded wait with failure propagation
4. **mlx-unauthenticated-network-listener**: Server binds 0.0.0.0 → bound to loopback
5. **mlx-model-setting-stale-client**: Model name captured at bean creation → update client config on model change
6. **mlx-remote-restart-local**: Remote endpoint restart launches local process → skip local restart for remote
7. **mlx-remote-inference-test-wrong-url**: Test uses default URL instead of configured → use configured endpoint
8. **mlx-manager-endpoint-mismatch**: Client and manager use different endpoints → aligned
9. **mlx-port-probe-false-positive**: Port check without health verification → verify MLX health endpoint
10. **mlx-start-client-timeout**: Dashboard 30s timeout vs 120s startup → increased timeout
11. **mlx-url-path-mismatch**: Missing `/v1` in URL → normalized paths
12. **mlx-startup-deadline-undercount**: Polling undercounts elapsed time → measure actual elapsed time
13. **mlx-remote-stop-local**: Remote stop kills local process → made remote stop a no-op
14. **mlx-local-port-mismatch**: Health check and start use different ports → derive port from configured endpoint
15. **settings-save-drops-gpuhub**: Unified save replaced gpuhub key with openai → preserved both
16. **mlx-stop-kills-unrelated-listener**: Stop kills any process on port → only stop verified MLX-owned
17. **mlx-restart-can-reuse-old-server**: Restart doesn't wait for old process exit → wait for process and port clear
18. **mlx-concurrent-start-hangs-after-failure**: Same as #3 → bounded wait
19. **mlx-health-check-adopts-unrelated-service**: Health check adopts any 200 response → establish MLX identity
20. **mlx-server-ignores-configured-model**: Server starts with hard-coded model → use configured model
21. **mlx-model-change-global-save-no-restart**: Global save bypasses restart → apply restart to save path
22. **mlx-audit-records-local-model**: Audit labels MLX with local model → read configured MLX model
23. **mlx-model-restart-interrupts-inference**: Restart doesn't check in-flight requests → defer restart
24. **mlx-test-inference-untracked**: Test inference not tracked → track with beginRequest/endRequest
25. **mlx-python-executable-name**: Uses `python` which may not exist → use `python3`
26. **mlx-endpoint-switch-leaves-local-process-running**: Endpoint change leaves local process → reconcile on change

## Why Deferred

The no-mistakes pipeline's review step uses the `pi` agent, which attempts to look up GPUHub documentation at `https://docs.gpuhub.com/best-practices/open-ports` during its review. This URL returns a 404 (the page does not exist on GPUHub's side). The pi agent fails with this 404, causing the pipeline run to fail at the review step. This happened twice (runs `01M3W1497WRR2TCH76FJDPAR2R` and `01M3WEJWCVFW4V9YGGF2D70QAC`).

The captain chose to defer rather than reroute the review to a different agent or skip the step.

## How to Resume

1. The branch `worktree-status-md` at `3e6e980fa249d22d30d73689ad4ea139a49b0255` is preserved with all fix commits
2. All tests pass (backend + frontend)
3. All review findings have been addressed
4. To resume: re-run `no-mistakes axi run` with the same intent once the pi agent's external dependency issue is resolved, or complete the remaining pipeline steps (test, document, lint, push, pr, ci) manually
5. The PR at https://github.com/kayis-rahman/swing-trade/pull/94 is not closed
