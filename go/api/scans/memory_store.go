package scans

import (
	"context"
	"sync"
	"time"
)

// MemoryRunStore is an in-memory RunRepository for tests and prototypes.
type MemoryRunStore struct {
	mu    sync.RWMutex
	runs  map[RunID]Run
	order []RunID
}

// NewMemoryRunStore returns a new in-memory store.
func NewMemoryRunStore() *MemoryRunStore {
	return &MemoryRunStore{runs: map[RunID]Run{}}
}

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
		if r.Status == StatusRunning || r.Status == StatusPaused {
			return r, true, nil
		}
	}
	return Run{}, false, nil
}

func (s *MemoryRunStore) UpdateStatus(_ context.Context, id RunID, status Status, completedAt *time.Time, errorMessage string) error {
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

func (s *MemoryRunStore) UpdateProgress(_ context.Context, id RunID, completed, failed int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	r.CompletedCount = completed
	r.FailedCount = failed
	s.runs[id] = r
	return nil
}
