package scans

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

// RunID identifies a candidate scan run.
type RunID string

// Status mirrors candidate scan lifecycle.
type Status string

// Status values for candidate scans.
const (
	StatusRunning   Status = "RUNNING"
	StatusPaused    Status = "PAUSED"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
	StatusFailed    Status = "FAILED"
)

// Run is the Go mirror of CandidateScanRunEntity.
type Run struct {
	RunID          RunID      `json:"runId"`
	Status         Status     `json:"status"`
	TotalSymbols   int        `json:"totalSymbols"`
	CompletedCount int        `json:"completedCount"`
	FailedCount    int        `json:"failedCount"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	StartedAt      time.Time  `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
}

// RunRepository is the persistence port for candidate scan runs.
type RunRepository interface {
	Create(ctx context.Context, run Run) error
	Get(ctx context.Context, id RunID) (Run, bool, error)
	List(ctx context.Context) ([]Run, error)
	FindActive(ctx context.Context) (Run, bool, error)
	UpdateStatus(ctx context.Context, id RunID, status Status, completedAt *time.Time, errorMessage string) error
	UpdateProgress(ctx context.Context, id RunID, completed, failed int) error
}

// ErrRunNotFound is returned for unknown scan runs.
var ErrRunNotFound = errors.New("candidate scan run not found")

// ErrAlreadyActive is returned when starting a scan while one is active.
var ErrAlreadyActive = errors.New("candidate scan already running")

// ErrIllegalTransition is returned on invalid state transitions.
var ErrIllegalTransition = errors.New("illegal candidate scan transition")

// CanTransition reports whether from -> to is allowed.
func CanTransition(from, to Status) bool {
	switch from {
	case StatusRunning:
		return to == StatusPaused || to == StatusCompleted || to == StatusCancelled || to == StatusFailed
	case StatusPaused:
		return to == StatusRunning || to == StatusCancelled || to == StatusFailed || to == StatusCompleted
	}
	return false
}

// Transition validates a state change.
func Transition(from, to Status) (Status, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, from, to)
	}
	return to, nil
}

// Manager coordinates pause/resume/cancel for candidate scans.
type Manager struct {
	Repo  RunRepository
	Clock clock.Clock

	mu        sync.Mutex
	active    *RunID
	paused    map[RunID]*bool
	cancelled map[RunID]*bool
}

// NewManager constructs a Manager.
func NewManager(repo RunRepository, clk clock.Clock) *Manager {
	if clk == nil {
		clk = clock.System{}
	}
	return &Manager{Repo: repo, Clock: clk, paused: map[RunID]*bool{}, cancelled: map[RunID]*bool{}}
}

func (m *Manager) now() time.Time {
	if m.Clock != nil {
		return m.Clock.Now()
	}
	return time.Now()
}

// Start registers a new running scan; persists it and marks active.
func (m *Manager) Start(ctx context.Context, totalSymbols int) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil {
		return Run{}, ErrAlreadyActive
	}
	if m.Repo != nil {
		if active, ok, err := m.Repo.FindActive(ctx); err == nil && ok {
			id := active.RunID
			m.active = &id
			if _, ok := m.paused[id]; !ok {
				paused := active.Status == StatusPaused
				m.paused[id] = &paused
			}
			if _, ok := m.cancelled[id]; !ok {
				cancelled := false
				m.cancelled[id] = &cancelled
			}
			return Run{}, ErrAlreadyActive
		}
	}
	run := Run{
		RunID:        RunID(fmt.Sprintf("scan-%d", m.now().UnixNano())),
		Status:       StatusRunning,
		TotalSymbols: totalSymbols,
		StartedAt:    m.now(),
	}
	paused := false
	cancelled := false
	if m.Repo != nil {
		if err := m.Repo.Create(ctx, run); err != nil {
			return Run{}, err
		}
	}
	m.active = &run.RunID
	m.paused[run.RunID] = &paused
	m.cancelled[run.RunID] = &cancelled
	return run, nil
}

// Pause pauses an active running scan.
func (m *Manager) Pause(ctx context.Context, id RunID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok, err := m.getLocked(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRunNotFound
	}
	if run.Status != StatusRunning {
		return fmt.Errorf("%w: cannot pause %s", ErrIllegalTransition, run.Status)
	}
	if _, err := Transition(run.Status, StatusPaused); err != nil {
		return err
	}
	now := m.now()
	if m.Repo != nil {
		if err := m.Repo.UpdateStatus(ctx, id, StatusPaused, nil, ""); err != nil {
			return err
		}
	}
	run.Status = StatusPaused
	if fl, ok := m.paused[id]; ok {
		*fl = true
	} else {
		tr := true
		m.paused[id] = &tr
	}
	_ = now
	return nil
}

func (m *Manager) getLocked(ctx context.Context, id RunID) (Run, bool, error) {
	if m.Repo != nil {
		return m.Repo.Get(ctx, id)
	}
	return Run{}, false, nil
}

// Resume resumes a paused scan.
func (m *Manager) Resume(ctx context.Context, id RunID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok, err := m.getLocked(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRunNotFound
	}
	if run.Status != StatusPaused {
		return fmt.Errorf("%w: cannot resume %s", ErrIllegalTransition, run.Status)
	}
	if _, err := Transition(run.Status, StatusRunning); err != nil {
		return err
	}
	if m.Repo != nil {
		if err := m.Repo.UpdateStatus(ctx, id, StatusRunning, nil, ""); err != nil {
			return err
		}
	}
	run.Status = StatusRunning
	if fl, ok := m.paused[id]; ok {
		*fl = false
	} else {
		fl := false
		m.paused[id] = &fl
	}
	return nil
}

// Cancel cancels a running or paused scan.
func (m *Manager) Cancel(ctx context.Context, id RunID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok, err := m.getLocked(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRunNotFound
	}
	if run.Status != StatusRunning && run.Status != StatusPaused {
		return fmt.Errorf("%w: cannot cancel %s", ErrIllegalTransition, run.Status)
	}
	if _, err := Transition(run.Status, StatusCancelled); err != nil {
		return err
	}
	now := m.now()
	if m.Repo != nil {
		if err := m.Repo.UpdateStatus(ctx, id, StatusCancelled, &now, "cancelled by request"); err != nil {
			return err
		}
	}
	run.Status = StatusCancelled
	if fl, ok := m.cancelled[id]; ok {
		*fl = true
	} else {
		tr := true
		m.cancelled[id] = &tr
	}
	if fl, ok := m.paused[id]; ok {
		*fl = false
	} else {
		fl := false
		m.paused[id] = &fl
	}
	if m.active != nil && *m.active == id {
		m.active = nil
	}
	return nil
}

// Complete marks scan completed.
func (m *Manager) Complete(ctx context.Context, id RunID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok, err := m.getLocked(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRunNotFound
	}
	if _, err := Transition(run.Status, StatusCompleted); err != nil {
		return err
	}
	now := m.now()
	if m.Repo != nil {
		if err := m.Repo.UpdateStatus(ctx, id, StatusCompleted, &now, ""); err != nil {
			return err
		}
	}
	run.Status = StatusCompleted
	if fl, ok := m.paused[id]; ok {
		*fl = false
	} else {
		fl := false
		m.paused[id] = &fl
	}
	if m.active != nil && *m.active == id {
		m.active = nil
	}
	return nil
}

// Fail marks scan failed.
func (m *Manager) Fail(ctx context.Context, id RunID, msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok, err := m.getLocked(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRunNotFound
	}
	if _, err := Transition(run.Status, StatusFailed); err != nil {
		return err
	}
	now := m.now()
	if m.Repo != nil {
		if err := m.Repo.UpdateStatus(ctx, id, StatusFailed, &now, msg); err != nil {
			return err
		}
	}
	run.Status = StatusFailed
	if fl, ok := m.paused[id]; ok {
		*fl = false
	} else {
		fl := false
		m.paused[id] = &fl
	}
	if m.active != nil && *m.active == id {
		m.active = nil
	}
	return nil
}

// IsPaused reports current pause flag (best-effort).
func (m *Manager) IsPaused(id RunID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fl, ok := m.paused[id]; ok {
		return *fl
	}
	return false
}

// IsCancelled reports current cancel flag.
func (m *Manager) IsCancelled(id RunID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fl, ok := m.cancelled[id]; ok {
		return *fl
	}
	return false
}

// Active returns active run id.
func (m *Manager) Active() *RunID {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return nil
	}
	id := *m.active
	return &id
}

// RecoverInterruptedRuns marks running/paused scans cancelled.
func (m *Manager) RecoverInterruptedRuns(ctx context.Context) (int, error) {
	if m.Repo == nil {
		return 0, nil
	}
	runs, err := m.Repo.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	now := m.now()
	for _, r := range runs {
		if r.Status == StatusRunning || r.Status == StatusPaused {
			_ = m.Repo.UpdateStatus(ctx, r.RunID, StatusCancelled, &now, "scan interrupted by API restart.")
			count++
		}
	}
	m.mu.Lock()
	m.active = nil
	m.paused = map[RunID]*bool{}
	m.cancelled = map[RunID]*bool{}
	m.mu.Unlock()
	return count, nil
}
