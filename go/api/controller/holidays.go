package controller

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

const (
	indiaTimeZone = "Asia/Kolkata"
	dateLayout    = "2006-01-02"
)

type holidayResponse struct {
	Date     string `json:"date"`
	Occasion string `json:"occasion"`
	Type     string `json:"type"`
}

// RegisterHolidayRoutes exposes the Java-compatible read-only holiday calendar.
func RegisterHolidayRoutes(mux *http.ServeMux, store data.HolidayReadStore, logger *slog.Logger, clk clock.Clock) {
	if clk == nil {
		clk = clock.System{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	mux.HandleFunc("GET /api/holidays/today", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeHolidayError(w, http.StatusServiceUnavailable, clk, "holiday data store is unavailable")
			return
		}
		location, err := time.LoadLocation(indiaTimeZone)
		if err != nil {
			writeHolidayInternalError(w, clk, r.URL.Path)
			return
		}
		today := clk.Now().In(location)
		date := calendarDate(today)
		holiday, err := store.GetHolidayByDate(r.Context(), date)
		if err != nil {
			logger.ErrorContext(r.Context(), "holiday lookup failed", "module", "api")
			writeHolidayInternalError(w, clk, r.URL.Path)
			return
		}
		closed := today.Weekday() == time.Saturday || today.Weekday() == time.Sunday || holiday != nil
		reason := map[string]any{}
		if closed {
			if holiday != nil {
				reason["occasion"] = holiday.Occasion
				reason["type"] = holiday.Type
			} else {
				reason["reason"] = "weekend"
			}
		}
		dataBody := map[string]any{
			"date": date.Format(dateLayout), "marketClosed": closed, "reason": reason,
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: dataBody, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
	mux.HandleFunc("GET /api/holidays", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeHolidayError(w, http.StatusServiceUnavailable, clk, "holiday data store is unavailable")
			return
		}
		location, err := time.LoadLocation(indiaTimeZone)
		if err != nil {
			writeHolidayError(w, http.StatusInternalServerError, clk, "Failed to list holidays: market timezone is unavailable")
			return
		}
		query := r.URL.Query()
		var rows []data.Holiday
		from, to := query.Get("from"), query.Get("to")
		if from != "" && to != "" {
			startDate, parseErr := time.Parse(dateLayout, from)
			if parseErr != nil {
				writeHolidayError(w, http.StatusInternalServerError, clk, "Failed to list holidays: "+parseErr.Error())
				return
			}
			endDate, parseErr := time.Parse(dateLayout, to)
			if parseErr != nil {
				writeHolidayError(w, http.StatusInternalServerError, clk, "Failed to list holidays: "+parseErr.Error())
				return
			}
			rows, err = store.ListHolidaysByDateRange(r.Context(), startDate, endDate)
		} else {
			rows, err = store.ListUpcomingHolidays(r.Context(), calendarDate(clk.Now().In(location)))
		}
		if err != nil {
			logger.ErrorContext(r.Context(), "holiday list failed", "module", "api")
			writeHolidayError(w, http.StatusInternalServerError, clk, "Failed to list holidays: "+err.Error())
			return
		}
		result := make([]holidayResponse, 0, len(rows))
		for _, row := range rows {
			result = append(result, holidayResponse{
				Date: row.Date.Format(dateLayout), Occasion: row.Occasion, Type: row.Type,
			})
		}
		payload := map[string]any{"count": len(result), "holidays": result}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: payload, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
}

func calendarDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func writeHolidayError(w http.ResponseWriter, status int, clk clock.Clock, message string) {
	writeResponseJSON(w, status, apiEnvelope{
		Data: nil, Error: stringValuePtr(message), Timestamp: javaLocalDateTime(clk.Now()),
	})
}

type holidayExceptionResponse struct {
	Code        string   `json:"code"`
	FieldErrors []string `json:"fieldErrors"`
	Message     string   `json:"message"`
	Path        string   `json:"path"`
	Status      int      `json:"status"`
	Timestamp   string   `json:"timestamp"`
}

func writeHolidayInternalError(w http.ResponseWriter, clk clock.Clock, path string) {
	writeResponseJSON(w, http.StatusInternalServerError, holidayExceptionResponse{
		Code: "INTERNAL_ERROR", FieldErrors: []string{},
		Message: "An unexpected error occurred. Please try again later.",
		Path:    path, Status: http.StatusInternalServerError,
		Timestamp: javaLocalDateTime(clk.Now()),
	})
}
