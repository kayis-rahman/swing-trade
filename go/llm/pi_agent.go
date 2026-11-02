package llm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PiAgentConfig mirrors Java.
type PiAgentConfig struct {
	Provider string
	Model    string
}

// PiAgentClient parity.
type PiAgentClient struct {
	Executable string
	Config     PiAgentConfig
	Timeout    time.Duration
}

const piAgentMaxOutputChars = 1_000_000

// NewPiAgentClient parity.
func NewPiAgentClient(executable, provider, model string, timeout time.Duration) *PiAgentClient {
	return &PiAgentClient{
		Executable: executable,
		Config:     PiAgentConfig{Provider: strings.TrimSpace(provider), Model: strings.TrimSpace(model)},
		Timeout:    timeout,
	}
}

// Configure parity.
func (c *PiAgentClient) Configure(provider, model string) {
	c.Config = PiAgentConfig{Provider: strings.TrimSpace(provider), Model: strings.TrimSpace(model)}
}

// GenerateChatCompletion parity.
func (c *PiAgentClient) GenerateChatCompletion(ctx context.Context, messages []map[string]string, _ int, _ float64) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prompt := buildPiAgentPrompt(messages)
	args := []string{
		c.Executable,
		"--provider", c.Config.Provider,
		"--model", c.Config.Model,
		"--print",
		"--no-tools",
		"--no-session",
		"--no-extensions",
		"--no-skills",
		"--no-prompt-templates",
		"--no-themes",
		"--no-approve",
		prompt,
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	cmd.Stdin = nil
	stdout, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("pi agent timed out after %s", timeout)
		}
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr == "" {
			stderr = err.Error()
		}
		return "", fmt.Errorf("pi agent failed: %s", stderr)
	}
	out := strings.TrimSpace(string(stdout))
	if out == "" {
		return "", fmt.Errorf("pi agent returned an empty response")
	}
	if len(out) > piAgentMaxOutputChars {
		out = out[:piAgentMaxOutputChars]
	}
	return out, nil
}

func buildPiAgentPrompt(messages []map[string]string) string {
	var b strings.Builder
	for _, m := range messages {
		role := strings.ToUpper(m["role"])
		if role == "" {
			role = "USER"
		}
		b.WriteString("[" + role + "]\n" + m["content"] + "\n\n")
	}
	return strings.TrimSpace(b.String())
}
