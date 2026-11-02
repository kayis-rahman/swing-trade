package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLayaClassify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sentiment":"positive","confidence":0.9}`))
	}))
	defer srv.Close()
	c := NewLayaClient(srv.URL, "key", 0)
	r, ok := c.Classify(context.Background(), "text")
	if !ok || r.Sentiment != "POSITIVE" || r.Confidence != 0.9 {
		t.Fatalf("got %+v ok=%v", r, ok)
	}
}

func TestLayaNotConfigured(t *testing.T) {
	c := NewLayaClient("", "key", 0)
	_, ok := c.Classify(context.Background(), "text")
	if ok {
		t.Fatal("expected false")
	}
}
