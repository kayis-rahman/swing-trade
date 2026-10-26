package data

import (
	"fmt"
	"testing"
	"time"
)

func TestParseFyersHistoryConvertsEpochToISTAndKeepsIntradayTime(t *testing.T) {
	stamp := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC).Unix()
	payload := fmt.Sprintf(`{"s":"ok","candles":[[%d,100.125,105,99,103.5,1200,0]]}`, stamp)
	candles, err := ParseFyersHistory([]byte(payload), "ABC", "15m")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("candles = %+v", candles)
	}
	candle := candles[0]
	if candle.Date.Format("2006-01-02") != "2026-10-23" || candle.BarTime.Format("15:04") != "09:30" {
		t.Fatalf("date/time = %s %s", candle.Date.Format("2006-01-02"), candle.BarTime.Format("15:04"))
	}
	if candle.Open.String() != "100.1250" || candle.AdjustedClose.String() != candle.Close.String() || candle.Volume != 1200 {
		t.Fatalf("unexpected candle: %+v", candle)
	}
}

func TestParseFyersHistoryNormalizesDailyTimeAndIgnoresErrorResponse(t *testing.T) {
	stamp := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC).Unix()
	payload := fmt.Sprintf(`{"s":"ok","candles":[[%d,10,12,9,11,5,0]]}`, stamp)
	candles, err := ParseFyersHistory([]byte(payload), "ABC", "D")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].BarTime.Hour() != 0 || candles[0].BarTime.Minute() != 0 {
		t.Fatalf("daily candles = %+v", candles)
	}
	empty, err := ParseFyersHistory([]byte(`{"s":"error","message":"invalid symbol"}`), "ABC", "D")
	if err != nil || len(empty) != 0 {
		t.Fatalf("error response = %v, %v", empty, err)
	}
}

func TestParseFyersHistoryRejectsMalformedPayloadAndSkipsIncompleteRows(t *testing.T) {
	if _, err := ParseFyersHistory([]byte("{"), "ABC", "D"); err == nil {
		t.Fatal("expected malformed JSON error")
	}
	candles, err := ParseFyersHistory([]byte(`{"s":"ok","candles":[[1,2],[1700000000,1,2,1,2,0,0]]}`), "ABC", "D")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 0 {
		t.Fatalf("invalid candles = %+v", candles)
	}
}
