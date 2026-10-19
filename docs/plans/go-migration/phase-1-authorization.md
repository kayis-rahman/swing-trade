# Phase 1 authorization and scope

Captain decision received 2026-10-17: Phase 0 and Gate G0 are waived. Do not collect more characterization evidence or write a G0 report. Existing committed and external evidence stays unchanged and may inform later parity tests.

Proceed with Phase 1, the Go platform skeleton, on branch `fm/swing-trade-go-migration-impl-v1`. Keep Java code and live services untouched. No merge, stage, tunnel, or Docker image push is authorized. Verification is local to this Mac. Captain separately authorized connections only to the development database at `192.168.0.100:5435`: use read-only sessions for reads; write tests must create and drop a unique `go_port_test_<random>` schema and never touch `public`. No migrations are allowed. Inbox 009 originally authorized runtime-only reads from the main project env file and prohibited copying it. Inbox 011 supersedes that restriction and authorizes the existing mode-600, gitignored `infra/env/.env` in this worktree. Read only `SPRING_DATASOURCE_URL`, `SPRING_DATASOURCE_USERNAME`, and `SPRING_DATASOURCE_PASSWORD`; never log or print values or copy the file elsewhere. The shell TCP probe to `192.168.0.100:5435` succeeds, but a Go process running directly on macOS reports `connect: no route to host` before establishing a session; no development-database schema or data was touched. Inbox 017 separately authorizes a fixed loopback-only throwaway Postgres target for local test verification when this network boundary blocks the dev DB.

The Go binary must expose infrastructure endpoints only, with no production traffic. The captain's local acceptance additionally requires one documented command to start the service, one command for unit/integration/static checks, and an end-to-end test that starts the real binary and exercises every route it exposes.

Phase 1 plan steps requiring deployment or operations on Pi/stage (run the skeleton on Pi and stage database backup/restore) remain deferred. No G0 report will be created.

## Authorization update

Captain instruction 051 (2026-10-19) authorizes adding a Go stage deployment set on this branch that can replace the Java API container. It does not authorize running that deployment, merging, touching production, or restarting the Pi host. A stage replacement must wait for the Phase 1 no-mistakes/evidence gate and the plan's stage backup/restore and safety prerequisites. The deployment set and its future prerequisites are tracked in [Go stage API replacement gate](stage-go-replacement.md).
