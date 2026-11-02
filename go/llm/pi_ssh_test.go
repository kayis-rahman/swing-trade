package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPiSSHHealthReady(t *testing.T) {
	m := NewPiSSHManager("u", "host", 8090, func(ctx context.Context, cmd string) (string, error) {
		return "ok", nil
	})
	if !m.HealthCheck(context.Background()) {
		t.Fatal("expected healthy")
	}
}

func TestPiSSHEndpoint(t *testing.T) {
	m := NewPiSSHManager("u", "host", 8090, func(ctx context.Context, cmd string) (string, error) {
		return "", errors.New("x")
	})
	m.Timeout = 200 * time.Millisecond
	if err := m.EnsureRunning(context.Background()); err == nil {
		t.Fatal("expected timeout/unready")
	}
	_ = m.Endpoint()
}
