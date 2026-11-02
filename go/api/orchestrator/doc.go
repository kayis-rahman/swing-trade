// Package orchestrator owns the Swing Trade job-pipeline orchestration primitives:
// run and stage domain types, the run/stage state machines, persistence ports, cancellation
// and interrupt bookkeeping, orphan-run reaping, run progress and summary assembly, and
// the run-summary notifier hook.
//
// Sibling slices (signal pipeline, candidate scans, SSE, notifications) program against the
// interfaces declared here rather than against any concrete storage or execution backend.
//
// This package intentionally contains no live provider, database, or broker calls; storage
// is injected through the port interfaces in store.go and execution through StageRunner.
package orchestrator
