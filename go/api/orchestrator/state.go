package orchestrator

import (
	"errors"
	"fmt"
)

// CanTransitionRun reports whether moving from -> to is allowed by the run state machine.
func CanTransitionRun(from, to RunStatus) bool {
	switch from {
	case RunStatusRunning:
		switch to {
		case RunStatusCompleted, RunStatusCompletedWithWarn, RunStatusFailed, RunStatusCancelled:
			return true
		}
	case RunStatusCompleted, RunStatusCompletedWithWarn, RunStatusFailed, RunStatusCancelled:
		// terminal: no further transition
	}
	return false
}

// TransitionRun validates and returns the target run status, erroring on illegal moves.
func TransitionRun(from, to RunStatus) (RunStatus, error) {
	if !CanTransitionRun(from, to) {
		return from, fmt.Errorf("illegal job run transition %s -> %s", from, to)
	}
	return to, nil
}

// CanTransitionStage reports whether moving from -> to is allowed.
func CanTransitionStage(from, to StageStatus) bool {
	switch from {
	case StageStatusPending:
		switch to {
		case StageStatusRunning, StageStatusSkipped, StageStatusCancelled:
			return true
		}
	case StageStatusRunning:
		switch to {
		case StageStatusCompleted, StageStatusDegraded, StageStatusError, StageStatusCancelled:
			return true
		}
	case StageStatusDegraded, StageStatusCompleted, StageStatusSkipped, StageStatusError, StageStatusCancelled:
		// terminal for a stage attempt
	}
	return false
}

// TransitionStage validates and returns target stage status.
func TransitionStage(from, to StageStatus) (StageStatus, error) {
	if !CanTransitionStage(from, to) {
		return from, fmt.Errorf("illegal job stage transition %s -> %s", from, to)
	}
	return to, nil
}

// IsTerminalRun reports whether status is terminal.
func IsTerminalRun(s RunStatus) bool {
	switch s {
	case RunStatusCompleted, RunStatusCompletedWithWarn, RunStatusFailed, RunStatusCancelled:
		return true
	}
	return false
}

// IsTerminalStage reports whether status is terminal.
func IsTerminalStage(s StageStatus) bool {
	switch s {
	case StageStatusCompleted, StageStatusDegraded, StageStatusSkipped, StageStatusError, StageStatusCancelled:
		return true
	}
	return false
}

// ErrIllegalTransition re-export for callers checking errors.Is.
var ErrIllegalTransition = errors.New("illegal state transition")
