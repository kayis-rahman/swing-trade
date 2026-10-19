# Go stage API replacement gate

## Scope and current status

The Go stage deployment set packages the existing no-traffic Go API as the
`swing-trade-stage-api` container on the existing stage network and port. It
does not connect to the stage database, run migrations, restart the Pi host, or
touch production. The dashboard and stage database containers are not replaced.

Replacing Java with the current Go skeleton makes Java-backed API endpoints
unavailable. Only Go health, admin-health, liveness, readiness, and Prometheus
routes are supported. The stage dashboard may therefore lose its trading and
orchestration functionality during this experiment. This is an explicit
stage-only behavior change, not a parity deployment.

The deployment set is implemented but **has not been run**. No stage host,
service, or database was accessed for this work. Stage replacement is blocked
until all prerequisites below are recorded and the Phase 1 no-mistakes and
evidence gate passes on the exact deployment commit.

## Preconditions before `./dev-stack.sh stage-go`

1. Run the no-mistakes Phase 1/evidence gate on the exact clean commit to be
deployed and record its run reference. The two existing review findings may
remain open under the captain's instruction; record them accurately and do not
represent them as resolved. They are not a separate blocker when the gate
passes.
2. Obtain separate authorization for a stage database backup and restore drill.
Create and checksum a stage backup, restore it to an isolated target, and
record both results. Do not restore over the live stage database. Phase 1's
existing backup/restore item is still deferred and is not satisfied by this
change.


3. Confirm the required stage `API_KEY`, the existing external Docker network,
and the target host architecture. The deploy script reads only `uname -m`,
passes `API_KEY` through Compose interpolation, and does not print the key.
4. Record the running Java container ID and image. The script preserves that
stopped container under a rollback name and leaves the dashboard and
database containers untouched.
5. Set `STAGE_GO_PHASE1_GATE_RECORD` and
`STAGE_GO_BACKUP_RESTORE_RECORD` to the completed JSON evidence records
described below.

Before any SSH call, `bin/deploy-go-stage.sh` validates both records. The Phase 1
record must contain exactly `kind: "no-mistakes-phase1"`,
`scope: "go-stage-replacement"`, the full current `commit`, `status: "passed"`,
and a nonempty `run_reference`. The backup record must contain exactly
`kind: "stage-backup-restore"`,
`scope: "stage-database-backup-restore"`, `status: "verified"`, nonempty
`backup_reference` and `restore_reference`, a 64-character lowercase hexadecimal
`backup_sha256`, `restore_target: "isolated"`, and `restore_verified: true`.
The commit must match the clean deployment HEAD. Missing, malformed, incomplete,
wrong-scope, or mismatched records refuse before contacting stage. These records
must reflect completed work; this documentation and local checks do not create
or satisfy them.

## Deployment and rollback

`./dev-stack.sh stage-go` cross-compiles for the stage host, packages the
executable into the Go image, and builds the image before stopping Java. It
then stops and renames only the Java API container, starts the Go container
with the same name, checks the documented Go health and metrics routes,
verifies admin health rejects requests without an API key, and confirms the
Prometheus target is up. It never stops PostgreSQL, the dashboard, or the Pi
host. If verification fails, it removes the Go container, restores the Java
container name, and starts Java again.

After a successful replacement, retain the stopped Java container until the
operator explicitly authorizes its removal. Verify the Go health routes and
Prometheus target. Do not run Java-specific feature smoke tests against the Go
skeleton. Do not call mutating or trading endpoints.

## Evidence boundary

Passing local Go tests and building the stage image prove the deployment
artifact only. They do not satisfy the stage backup/restore drill, authorize a
stage replacement, or establish production readiness. No stage deployment,
merge, production operation, or Pi restart is authorized by this plan.
