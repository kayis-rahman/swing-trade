package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteSSEEmitsEventAndDataFrames(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteSSE(recorder, "SIGNAL_DONE", `{"symbol":"INFY"}`)

	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control = %q, want no-cache", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event: SIGNAL_DONE\n") {
		t.Fatalf("body = %q, want the event frame", body)
	}
	if !strings.Contains(body, `data: {"symbol":"INFY"}`+"\n\n") {
		t.Fatalf("body = %q, want the terminated data frame", body)
	}
	if !recorder.Flushed {
		t.Fatal("WriteSSE did not flush the response")
	}
}

func TestHealthExtraRespondsWithoutContent(t *testing.T) {
	recorder := httptest.NewRecorder()
	HealthExtra(recorder, httptest.NewRequest(http.MethodGet, "/api/health/extra", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}
