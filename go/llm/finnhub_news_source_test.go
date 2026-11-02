package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFinnhubNewsSourceParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"headline":"TCS launches platform","url":"https://news.example/tcs","summary":"` + repeatX(1001) + `","datetime":1726394400},{"headline":"TCS launches platform"}]`))
	}))
	defer srv.Close()

	src := NewFinnhubNewsSource("key", 10, srv.Client())
	// override baseURL via field not exported; instead create with client and call direct URL by temp hack: set base via reflection impossible. Use localhost request by manually? Simpler: create source with default and monkey? Instead test parse by setting base URL field using package access.
	src.baseURL = srv.URL
	arts, err := src.Fetch(context.Background(), "TCS")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("got %d", len(arts))
	}
	if len(arts[0].Description) != 1000 {
		t.Fatalf("desc len %d", len(arts[0].Description))
	}
}

func TestFinnhubNoKey(t *testing.T) {
	src := NewFinnhubNewsSource("", 10, nil)
	arts, err := src.Fetch(context.Background(), "TCS")
	if err != nil || arts != nil {
		t.Fatalf("expected nil,nil got %v %v", arts, err)
	}
}

func repeatX(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
