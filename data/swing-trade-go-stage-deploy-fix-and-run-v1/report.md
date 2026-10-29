# Swing Trade Go stage deploy fix and run

## Fix

`bin/verify-go-stage-authority.sh` Phase 1 gate cross-build branch now uses an explicit throwaway output path:

```bash
tmpbin="$(mktemp)"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o "$tmpbin" ./go/api/cmd/swingtrade-api
rm -f "$tmpbin"
```

No `swingtrade-api` binary is written to the repository root. Clean-worktree guard and refusal behavior are unchanged.

## Validation

- `bash -n bin/verify-go-stage-authority.sh` passed.
- `bin/verify-go-stage-authority.sh` passed: Phase 1 gate, stage snapshot, and stage restore.
- `git status --porcelain` was empty after the verifier; no `swingtrade-api` artifact appeared.

## Stage run

Command: `./dev-stack.sh stage-go` (stage-only, pi-node/piworm.local, port 8081, never production).

Result: stage deploy was refused after Go image build and Go API smoke checks, during the authoritative Prometheus target check.

Verifier output:

```text
Authoritative stage access verifier passed: Phase 1 gate, stage snapshot, and stage restore.
Building Go API for Linux/arm64 at 90d992ff111e.
```

Go API smoke checks passed:

```text
Go API stage smoke check passed: /api/health
Go API stage smoke check passed: /actuator/health/liveness
Go API stage smoke check passed: /actuator/health/readiness
Go API stage smoke check passed: /actuator/prometheus
Go API stage smoke check passed: admin API key is required.
```

Final refusal:

```text
Prometheus did not report the swing-trade-stage target as up.
Go stage health verification failed; restoring the stopped Java API container.
```

Live Prometheus target evidence during the attempt showed DNS/connection failure for `swing-trade-stage-api:8080`:

```text
Get "http://swing-trade-stage-api:8080/actuator/prometheus": dial tcp: lookup swing-trade-stage-api on 127.0.0.11:53: no such host
Get "http://swing-trade-stage-api:8080/actuator/prometheus": dial tcp 172.21.0.4:8080: connect: connection refused
```

The deploy script rolled back and restored the Java stage API on port 8081. Go stage URL `http://piworm.local:8081/...` was not left live after the refused deploy.

## Worktree

Worktree remains clean.
