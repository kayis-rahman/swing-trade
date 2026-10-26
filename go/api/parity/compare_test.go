package parity

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

//go:embed testdata/watchlist-all-java.json
var watchlistJavaFixture []byte

//go:embed testdata/watchlist-all-go.json
var watchlistGoFixture []byte

func TestConfiguredReadParity(t *testing.T) {
	javaBaseURL := os.Getenv("JAVA_SERVICE_URL")
	if javaBaseURL == "" {
		javaBaseURL = os.Getenv("PARITY_JAVA_BASE_URL")
	}
	goBaseURL := os.Getenv("GO_SERVICE_URL")
	if goBaseURL == "" {
		goBaseURL = os.Getenv("PARITY_GO_BASE_URL")
	}
	if javaBaseURL == "" || goBaseURL == "" {
		t.Skip("set JAVA_SERVICE_URL and GO_SERVICE_URL to compare running services")
	}
	differences, err := Compare(http.DefaultClient, javaBaseURL, goBaseURL, []Request{
		{Path: "/api/health"},
		{Path: "/api/holidays"},
		{Path: "/api/holidays/today"},
		{Path: "/api/watchlist"},
		{Path: "/api/watchlist/all"},
		{Path: "/api/settings"},
		{Path: "/api/settings/discord"},
		{Path: "/api/settings/gpuhub"},
		{Path: "/api/settings/trading"},
		{Path: "/api/settings/scanning"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, difference := range differences {
		t.Errorf("%s: %s", difference.Path, difference.Reason)
	}
}

func TestCompareWatchlistAllMatchesRecordsByID(t *testing.T) {
	java := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(watchlistJavaFixture)
	}))
	defer java.Close()
	goServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(watchlistGoFixture)
	}))
	defer goServer.Close()

	differences, err := Compare(java.Client(), java.URL, goServer.URL, []Request{{Path: "/api/watchlist/all"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(differences) != 0 {
		t.Fatalf("unexpected differences: %+v", differences)
	}
}

func TestCompareReportsStatusHeaderAndBodyDifferences(t *testing.T) {
	cases := []struct {
		name       string
		java       response
		goResponse response
		want       string
	}{
		{"status", response{status: 200}, response{status: 503}, "status"},
		{"content type", response{status: 200, contentType: "application/json"}, response{status: 200, contentType: "text/plain"}, "content-type"},
		{"body", response{status: 200, contentType: "application/json", body: []byte(`{"ok":true}`)}, response{status: 200, contentType: "application/json", body: []byte(`{"ok":false}`)}, "response bodies"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := compareResponses("/api/example", test.java, test.goResponse)
			if got == nil || !contains(got.Reason, test.want) {
				t.Fatalf("difference = %+v, want reason containing %q", got, test.want)
			}
		})
	}
}

func TestRecordsByIDRejectsInvalidEnvelopes(t *testing.T) {
	for _, value := range []any{nil, []any{}, map[string]any{"data": "not an array"}, map[string]any{"data": []any{map[string]any{"symbol": "ABC"}}}} {
		if _, ok := recordsByID(value); ok {
			t.Errorf("recordsByID(%#v) unexpectedly succeeded", value)
		}
	}
}

func TestEqualJSONValueComparesNestedValues(t *testing.T) {
	left := map[string]any{"data": []any{map[string]any{"ok": true}}}
	right := map[string]any{"data": []any{map[string]any{"ok": true}}}
	if !reflect.DeepEqual(left, right) || !equalJSONValue(left, right) {
		t.Fatal("equal nested JSON values should compare equal")
	}
	right["extra"] = true
	if equalJSONValue(left, right) {
		t.Fatal("different nested JSON values should not compare equal")
	}
}

func contains(value, substring string) bool {
	for index := 0; index+len(substring) <= len(value); index++ {
		if value[index:index+len(substring)] == substring {
			return true
		}
	}
	return false
}
