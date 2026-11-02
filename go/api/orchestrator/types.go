package orchestrator

import (
	"context"
	"errors"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

// RunID is the opaque identifier of an orchestration run.
type RunID string

// TriggerType mirrors Java JobRun.TriggerType.
type TriggerType string

const (
	TriggerManual    TriggerType = "MANUAL"
	TriggerScheduled TriggerType = "SCHEDULED"
)

// RunStatus mirrors Java JobRun.Status.
type RunStatus string

const (
	RunStatusRunning           RunStatus = "RUNNING"
	RunStatusCompleted         RunStatus = "COMPLETED"
	RunStatusCompletedWithWarn RunStatus = "COMPLETED_WITH_WARNINGS"
	RunStatusFailed            RunStatus = "FAILED"
	RunStatusCancelled         RunStatus = "CANCELLED"
)

// StageName mirrors Java JobRunStage.StageName; declaration order drives the
// PENDING rows created up front and must stay in sync with execution order.
type StageName string

const (
	StageDataFetch   StageName = "DATA_FETCH"
	StageSignal      StageName = "SIGNAL"
	StageBacktest    StageName = "BACKTEST"
	StageNews        StageName = "NEWS"
	StageSentiment   StageName = "SENTIMENT"
	StageLLMAnalysis StageName = "LLM_ANALYSIS"
	StagePaperTrade  StageName = "PAPER_TRADE"
)

// AllStageNames returns the canonical execution order.
func AllStageNames() []StageName {
	return []StageName{
		StageDataFetch, StageSignal, StageBacktest, StageNews,
		StageSentiment, StageLLMAnalysis, StagePaperTrade,
	}
}

// StageStatus mirrors Java JobRunStage.Status.
type StageStatus string

const (
	StageStatusPending   StageStatus = "PENDING"
	StageStatusRunning   StageStatus = "RUNNING"
	StageStatusCompleted StageStatus = "COMPLETED"
	StageStatusDegraded  StageStatus = "DEGRADED"
	StageStatusSkipped   StageStatus = "SKIPPED"
	StageStatusError     StageStatus = "ERROR"
	StageStatusCancelled StageStatus = "CANCELLED"
)

// Run is the Go mirror of Java JobRun (immutable value).
type Run struct {
	RunID          RunID       `json:"runId"`
	TriggerType    TriggerType `json:"triggerType"`
	Status         RunStatus   `json:"status"`
	StartedAt      time.Time   `json:"startedAt"`
	CompletedAt    *time.Time  `json:"completedAt,omitempty"`
	SymbolsCount   int         `json:"symbolsCount"`
	CompletedCount int         `json:"completedCount"`
	FailedCount    int         `json:"failedCount"`
	ErrorMessage   string      `json:"errorMessage,omitempty"`
}

// Stage is the Go mirror of Java JobRunStage.
type Stage struct {
	RunID         RunID       `json:"runId"`
	Symbol        string      `json:"symbol"`
	StageName     StageName   `json:"stageName"`
	Status        StageStatus `json:"status"`
	StartedAt     *time.Time  `json:"startedAt,omitempty"`
	CompletedAt   *time.Time  `json:"completedAt,omitempty"`
	DurationMs    *int64      `json:"durationMs,omitempty"`
	ErrorMessage  string      `json:"errorMessage,omitempty"`
	LogDetails    string      `json:"logDetails,omitempty"`
	ResultSummary string      `json:"resultSummary,omitempty"`
	Details       string      `json:"details,omitempty"`
}

// RunRequest carries caller-supplied scoping for a run.
type RunRequest struct {
	Symbols []string
}

// StageExecutionResult is the value returned by a StageRunner.
type StageExecutionResult struct {
	Status       StageStatus `json:"status"`
	Summary      string      `json:"summary,omitempty"`
	Details      string      `json:"details,omitempty"`
	ReasonCode   string      `json:"reasonCode,omitempty"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
}

// ConcurrentRunError is returned when a run is started while another is active.
type ConcurrentRunError struct {
	ActiveRun Run
}

func (e *ConcurrentRunError) Error() string { return "another job run is already active" }

// ErrRunNotFound is returned for unknown run ids.
var ErrRunNotFound = errors.New("job run not found")

// RunRepository is the persistence port for JobRun values.
type RunRepository interface {
	Create(ctx context.Context, run Run) error
	Get(ctx context.Context, id RunID) (Run, bool, error)
	List(ctx context.Context) ([]Run, error)
	FindActive(ctx context.Context) (Run, bool, error)
	UpdateStatus(ctx context.Context, id RunID, status RunStatus, completedAt *time.Time, errorMessage string) error
	IncrementCompleted(ctx context.Context, id RunID) error
}

// StageRepository is the persistence port for JobRunStage values.
type StageRepository interface {
	CreateBatch(ctx context.Context, stages []Stage) error
	ListByRun(ctx context.Context, id RunID) ([]Stage, error)
	Update(ctx context.Context, stage Stage) error
}

// RunSummaryNotifier is the hook invoked when a run reaches a terminal state.
type RunSummaryNotifier interface {
	Notify(ctx context.Context, summary RunSummary) error
}

// Clock is re-exported convenience for orchestrator consumers.
type Clock = clock.Clock

// StageRunner executes a single stage for a symbol within a run.
// Sibling slices supply their own runners; the orchestrator calls them.
type StageRunner func(ctx context.Context, run Run, stage Stage) StageExecutionResult
