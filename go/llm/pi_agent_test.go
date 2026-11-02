package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPiAgentSuccess(t *testing.T) {
	f, _ := os.CreateTemp("", "pi-*")
	_ = f.Close()
	_ = os.WriteFile(f.Name(), []byte("#!/bin/sh\necho '{\"ok\":true}'"), 0755) //nolint:gosec
	_ = os.Chmod(f.Name(), 0755)                                                //nolint:gosec
	defer func() { _ = os.Remove(f.Name()) }()
	c := NewPiAgentClient(f.Name(), "prov", "model", time.Second)
	out, err := c.GenerateChatCompletion(context.Background(), []map[string]string{{"role": "user", "content": "hi"}}, 10, 0)
	if err != nil || out != "{\"ok\":true}" {
		t.Fatalf("got %q err %v", out, err)
	}
}

func TestPiAgentTimeout(t *testing.T) {
	f, _ := os.CreateTemp("", "pi-*")
	_ = f.Close()
	_ = os.WriteFile(f.Name(), []byte("#!/bin/sh\nsleep 2\necho done"), 0755) //nolint:gosec
	_ = os.Chmod(f.Name(), 0755)                                              //nolint:gosec
	defer func() { _ = os.Remove(f.Name()) }()
	c := NewPiAgentClient(f.Name(), "prov", "model", 100*time.Millisecond)
	_, err := c.GenerateChatCompletion(context.Background(), []map[string]string{{"role": "user", "content": "hi"}}, 10, 0)
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestPiAgentEmpty(t *testing.T) {
	f, _ := os.CreateTemp("", "pi-*")
	_ = f.Close()
	_ = os.WriteFile(f.Name(), []byte("#!/bin/sh\n"), 0755) //nolint:gosec
	_ = os.Chmod(f.Name(), 0755)                            //nolint:gosec
	defer func() { _ = os.Remove(f.Name()) }()
	c := NewPiAgentClient(f.Name(), "prov", "model", time.Second)
	_, err := c.GenerateChatCompletion(context.Background(), []map[string]string{{"role": "user", "content": "hi"}}, 10, 0)
	if err == nil {
		t.Fatal("expected empty error")
	}
}
