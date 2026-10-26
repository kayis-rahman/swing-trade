package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeHolidayReader struct {
	holiday  *data.Holiday
	rows     []data.Holiday
	err      error
	date     time.Time
	from     time.Time
	to       time.Time
	upcoming bool
}

func (f *fakeHolidayReader) GetHolidayByDate(_ context.Context, date time.Time) (*data.Holiday, error) {
	f.date = date
	return f.holiday, f.err
}

func (f *fakeHolidayReader) ListHolidaysByDateRange(_ context.Context, from, to time.Time) ([]data.Holiday, error) {
	f.from, f.to = from, to
	return f.rows, f.err
}

func (f *fakeHolidayReader) ListUpcomingHolidays(_ context.Context, from time.Time) ([]data.Holiday, error) {
	f.from, f.upcoming = from, true
	return f.rows, f.err
}

func TestHolidayReadEndpointsMatchJavaResponses(t *testing.T) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, ist) // Saturday
	clk := fixedClock{now: now}
	holidayDate := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	reader := &fakeHolidayReader{rows: []data.Holiday{{Date: holidayDate, Occasion: "Diwali", Type: "FULL"}}}
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, reader, slog.New(slog.NewTextHandler(io.Discard, nil)), clk)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays", nil))
	if response.Code != http.StatusOK || !reader.upcoming || !reader.from.Equal(time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("upcoming holidays = status %d, reader %#v, body %s", response.Code, reader, response.Body.String())
	}
	var listed struct {
		Data struct {
			Count    int `json:"count"`
			Holidays []struct {
				Date     string `json:"date"`
				Occasion string `json:"occasion"`
				Type     string `json:"type"`
			} `json:"holidays"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Data.Count != 1 || len(listed.Data.Holidays) != 1 || listed.Data.Holidays[0].Date != "2026-10-26" || listed.Data.Holidays[0].Type != "FULL" {
		t.Fatalf("holiday list = %#v", listed.Data)
	}

	reader.upcoming = false
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays?from=2026-10-01&to=2026-10-31", nil))
	if response.Code != http.StatusOK || reader.upcoming || !reader.from.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) || !reader.to.Equal(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("range holidays = status %d, reader %#v, body %s", response.Code, reader, response.Body.String())
	}
	var ranged struct {
		Data struct {
			Count    int `json:"count"`
			Holidays []struct {
				Date string `json:"date"`
			} `json:"holidays"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &ranged); err != nil {
		t.Fatal(err)
	}
	if ranged.Data.Count != 1 || len(ranged.Data.Holidays) != 1 || ranged.Data.Holidays[0].Date != "2026-10-26" {
		t.Fatalf("ranged holiday payload = %#v", ranged.Data)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays/today", nil))
	var today struct {
		Data struct {
			Date         string         `json:"date"`
			MarketClosed bool           `json:"marketClosed"`
			Reason       map[string]any `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &today); err != nil {
		t.Fatal(err)
	}
	if today.Data.Date != "2026-10-24" || !today.Data.MarketClosed || today.Data.Reason["reason"] != "weekend" {
		t.Fatalf("Saturday market status = %#v", today.Data)
	}
	if !reader.date.Equal(time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("today lookup date = %s", reader.date)
	}
}

func TestHolidayTodayReturnsHolidayReasonOnWeekday(t *testing.T) {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Date(2026, 10, 26, 12, 0, 0, 0, ist)
	reader := &fakeHolidayReader{holiday: &data.Holiday{Date: today, Occasion: "Diwali", Type: "FULL"}}
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, reader, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: today})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays/today", nil))
	var body struct {
		Data struct {
			MarketClosed bool           `json:"marketClosed"`
			Reason       map[string]any `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Data.MarketClosed || body.Data.Reason["occasion"] != "Diwali" || body.Data.Reason["type"] != "FULL" {
		t.Fatalf("holiday reason = %#v", body.Data)
	}
}

func TestHolidayReadWithoutStoreReturnsServiceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, nil, nil, fixedClock{now: time.Now()})
	for _, path := range []string{"/api/holidays", "/api/holidays/today"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s without store = %d, want 503", path, response.Code)
		}
	}
}

func TestHolidayReadErrorsUseJavaApiResponse(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, &fakeHolidayReader{err: errors.New("database failed")}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays", nil))
	var body struct {
		Data    any    `json:"data"`
		Error   string `json:"error"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body.Success || body.Data != nil || body.Error != "Failed to list holidays: database failed" {
		t.Fatalf("holiday failure response = %d %#v", response.Code, body)
	}
}

func TestHolidayTodayDatabaseErrorUsesGlobalErrorContract(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, &fakeHolidayReader{err: errors.New("database failed")}, nil, fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays/today", nil))
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body["code"] != "INTERNAL_ERROR" || body["path"] != "/api/holidays/today" || body["message"] != "An unexpected error occurred. Please try again later." {
		t.Fatalf("holiday today error response = %d %#v", response.Code, body)
	}
}

func TestHolidayListInvalidDateUsesApiResponseErrorEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHolidayRoutes(mux, &fakeHolidayReader{}, nil, fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/holidays?from=bad&to=2026-10-31", nil))
	var body struct {
		Data    any    `json:"data"`
		Error   string `json:"error"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body.Success || body.Data != nil || !strings.HasPrefix(body.Error, "Failed to list holidays:") {
		t.Fatalf("invalid holiday range response = %d %#v", response.Code, body)
	}
}

var _ coreclock.Clock = fixedClock{}
var _ data.HolidayReadStore = (*fakeHolidayReader)(nil)
