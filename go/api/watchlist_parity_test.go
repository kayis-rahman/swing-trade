package api

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// The Java findAll() contract does not guarantee ordering for /api/watchlist/all.
// Compare complete records by id so database row order is not treated as a diff.
func watchlistRecordsByID(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var envelope struct {
		Data []map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	id := func(record map[string]any) int64 {
		value, err := strconv.ParseInt(record["id"].(json.Number).String(), 10, 64)
		if err != nil {
			t.Fatalf("invalid watchlist id: %v", err)
		}
		return value
	}
	sort.Slice(envelope.Data, func(i, j int) bool {
		return id(envelope.Data[i]) < id(envelope.Data[j])
	})
	return envelope.Data
}

func TestWatchlistAllParityIgnoresRecordOrderByID(t *testing.T) {
	java := []byte(`{"data":[{"id":2,"symbol":"TCS"},{"id":1,"symbol":"RELIANCE"}]}`)
	goResponse := []byte(`{"data":[{"id":1,"symbol":"RELIANCE"},{"id":2,"symbol":"TCS"}]}`)
	if !reflect.DeepEqual(watchlistRecordsByID(t, java), watchlistRecordsByID(t, goResponse)) {
		t.Fatal("equivalent /api/watchlist/all records differ after matching by id")
	}
}

func TestWatchlistAllParityStillDetectsRecordDifferences(t *testing.T) {
	java := []byte(`{"data":[{"id":1,"symbol":"RELIANCE"}]}`)
	goResponse := []byte(`{"data":[{"id":1,"symbol":"OTHER"}]}`)
	if reflect.DeepEqual(watchlistRecordsByID(t, java), watchlistRecordsByID(t, goResponse)) {
		t.Fatal("record content difference was ignored")
	}
}
