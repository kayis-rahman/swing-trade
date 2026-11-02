package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()
	c := NewClientWithModel(srv.URL, "key", "m")
	got, err := c.GenerateChatCompletion(context.Background(), []map[string]string{{"role": "user", "content": "x"}}, 10, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hi" {
		t.Fatalf("got %q", got)
	}
}
