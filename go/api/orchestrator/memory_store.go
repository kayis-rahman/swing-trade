package orchestrator

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MemoryRunStore is an in-memory RunRepository for tests and prototypes.
type MemoryRunStore struct {
	mu    sync.RWMutex
	runs  map[RunID]Run
	order []RunID
}

func NewMemoryRunStore() *MemoryRunStore { return &MemoryRunStore{runs: map[RunID]Run{}} }

func (s *MemoryRunStore) Create(_ context.Context, run Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.RunID] = run
	s.order = append(s.order, run.RunID)
	return nil
}
func (s *MemoryRunStore) Get(_ context.Context, id RunID) (Run, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	return r, ok, nil
}
func (s *MemoryRunStore) List(_ context.Context) ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Run, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.runs[id])
	}
	return out, nil
}
func (s *MemoryRunStore) FindActive(_ context.Context) (Run, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.order {
		r := s.runs[id]
		if r.Status == RunStatusRunning {
			return r, true, nil
		}
	}
	return Run{}, false, nil
}
func (s *MemoryRunStore) UpdateStatus(_ context.Context, id RunID, status RunStatus, completedAt *time.Time, errorMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	r.Status = status
	if completedAt != nil {
		t := *completedAt
		r.CompletedAt = &t
	}
	r.ErrorMessage = errorMessage
	s.runs[id] = r
	return nil
}
func (s *MemoryRunStore) IncrementCompleted(_ context.Context, id RunID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	r.CompletedCount++
	s.runs[id] = r
	return nil
}

// MemoryStageStore is an in-memory StageRepository.
type MemoryStageStore struct {
	mu     sync.RWMutex
	stages []Stage
}

func NewMemoryStageStore() *MemoryStageStore { return &MemoryStageStore{} }

func (s *MemoryStageStore) CreateBatch(_ context.Context, stages []Stage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stages = append(s.stages, stages...)
	return nil
}
func (s *MemoryStageStore) ListByRun(_ context.Context, id RunID) ([]Stage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Stage
	for _, st := range s.stages {
		if st.RunID == id {
			out = append(out, st)
		}
	}
	return out, nil
}
func (s *MemoryStageStore) Update(_ context.Context, stage Stage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, st := range s.stages {
		if st.RunID == stage.RunID && st.Symbol == stage.Symbol && st.StageName == stage.StageName {
			s.stages[i] = stage
			return nil
		}
	}
	return errors.New("stage not found")
}
