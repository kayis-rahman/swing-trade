package llm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SSHRunner runs a remote command.
// SSHRunner parity.
type SSHRunner func(ctx context.Context, remoteCmd string) (string, error)

// PiSSHManager manages llama.cpp on a remote Pi over SSH.
// PiSSHManager parity.
type PiSSHManager struct {
	SSHUser     string
	SSHHost     string
	Port        int
	ModelPath   string
	Threads     int
	ContextSize int
	Timeout     time.Duration
	Runner      SSHRunner
	running     bool
}

// NewPiSSHManager parity.
func NewPiSSHManager(user, host string, port int, runner SSHRunner) *PiSSHManager {
	if runner == nil {
		runner = defaultSSHRunner(user, host)
	}
	return &PiSSHManager{SSHUser: user, SSHHost: host, Port: port, Timeout: 45 * time.Second, Runner: runner, Threads: 2, ContextSize: 4096}
}

func defaultSSHRunner(user, host string) SSHRunner {
	return func(ctx context.Context, remoteCmd string) (string, error) {
		cmd := exec.CommandContext(ctx, "ssh", fmt.Sprintf("%s@%s", user, host), remoteCmd) //nolint:gosec
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
}

func (m *PiSSHManager) endpoint() string {
	return fmt.Sprintf("http://%s:%d", m.SSHHost, m.Port)
}

// Endpoint parity.
func (m *PiSSHManager) Endpoint() string { return m.endpoint() }

// HealthCheck parity.
func (m *PiSSHManager) HealthCheck(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := m.Runner(ctx, fmt.Sprintf("curl -fsS http://127.0.0.1:%d/health", m.Port))
	if err != nil {
		return false
	}
	return strings.Contains(out, "ok") || strings.TrimSpace(out) != ""
}

// EnsureRunning parity.
func (m *PiSSHManager) EnsureRunning(ctx context.Context) error {
	if m.HealthCheck(ctx) {
		m.running = true
		return nil
	}
	startCtx, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()
	cmd := fmt.Sprintf("nohup llama-server --model %s --port %d --threads %d --ctx-size %d >/tmp/llama.log 2>&1 &", shellQuote(m.ModelPath), m.Port, m.Threads, m.ContextSize)
	_, _ = m.Runner(startCtx, cmd)
	deadline := time.Now().Add(m.Timeout)
	for time.Now().Before(deadline) {
		if m.HealthCheck(ctx) {
			m.running = true
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("llama-server did not become ready on %s:%d", m.SSHHost, m.Port)
}

// Stop parity.
func (m *PiSSHManager) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()
	_, err := m.Runner(ctx, fmt.Sprintf("pkill -f 'llama-server.*--port %d' || true", m.Port))
	m.running = false
	return err
}

// IsRunning parity.
func (m *PiSSHManager) IsRunning() bool { return m.running || m.HealthCheck(context.Background()) }

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
