package api

import (
	"fmt"
	"net/http"
)

// WriteSSE writes one server-sent event and flushes it to the client.
func WriteSSE(w http.ResponseWriter, event string, data string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
