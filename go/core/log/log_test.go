package log

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewWritesStructuredModuleField(t *testing.T) {
	var output bytes.Buffer
	New(&output, slog.LevelInfo, "api").Info("request handled", "traceId", "abcd1234", "symbol", "TCS")
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON handler emitted invalid JSON: %v", err)
	}
	for key, want := range map[string]any{"module": "api", "traceId": "abcd1234", "symbol": "TCS", "msg": "request handled"} {
		if got[key] != want {
			t.Errorf("field %q = %#v, want %#v", key, got[key], want)
		}
	}
}
