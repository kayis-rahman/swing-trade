package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

const memoryMegabyte = 1024 * 1024

type healthResponse struct {
	Components map[string]healthComponent `json:"components"`
	Status     string                     `json:"status"`
	SystemInfo healthSystemInfo           `json:"systemInfo"`
	Timestamp  string                     `json:"timestamp"`
}

type healthComponent struct {
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	Description    string         `json:"description"`
	ResponseTimeMS *int64         `json:"responseTimeMs"`
	Details        map[string]any `json:"details"`
}

// healthSystemInfo retains Java-era wire keys for client compatibility through Phase 8.
// javaVersion maps to runtime.Version; heap fields map to HeapIdle-HeapReleased, HeapSys, and HeapAlloc.
// Phase 8 may rename the fields to Go-specific names after the compatibility window.
type healthSystemInfo struct {
	JavaVersion         string `json:"javaVersion"`
	OSName              string `json:"osName"`
	OSVersion           string `json:"osVersion"`
	Architecture        string `json:"architecture"`
	AvailableProcessors int    `json:"availableProcessors"`
	FreeMemoryMB        int64  `json:"freeMemoryMB"`
	MaxMemoryMB         int64  `json:"maxMemoryMB"`
	TotalMemoryMB       int64  `json:"totalMemoryMB"`
}

// RegisterHealthRoutes installs the read-only Java-compatible health endpoints.
func RegisterHealthRoutes(mux *http.ServeMux, dataConfigured bool, clk clock.Clock, startedAt time.Time) {
	if clk == nil {
		clk = clock.System{}
	}
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, clk, healthResponse{Status: "UP", Components: map[string]healthComponent{
			"api": component("api", "UP", "API Service", nil),
		}})
	})
	mux.HandleFunc("GET /api/health/details", func(w http.ResponseWriter, r *http.Request) {
		components := map[string]healthComponent{
			"api":      component("api", "UP", "REST API Service", nil),
			"data":     component("data", "UP", dataDescription(dataConfigured), dataDetails(dataConfigured)),
			"strategy": component("strategy", "UP", "Strategy Service (not configured)", nil),
			"llm":      component("llm", "UP", "LLM Service (not configured)", nil),
		}
		if queryBool(r.URL.Query().Get("verbose")) {
			for name, value := range components {
				value.Details["javaVersion"] = runtime.Version()
				value.Details["osName"] = runtime.GOOS
				value.Details["uptime"] = formatUptime(clk.Now().Sub(startedAt))
				components[name] = value
			}
		}
		writeHealth(w, clk, healthResponse{Status: "UP", Components: components})
	})
	mux.HandleFunc("GET /api/health/database", func(w http.ResponseWriter, _ *http.Request) {
		var database healthComponent
		if dataConfigured {
			database = component("database", "UP", "Database Connection", map[string]any{
				"type": "PostgreSQL", "version": "PostgreSQL 14+",
			})
		} else {
			database = component("database", "UP", "Database Service (not configured)", nil)
		}
		writeHealth(w, clk, healthResponse{Status: "UP", Components: map[string]healthComponent{"database": database}})
	})
	mux.HandleFunc("GET /api/health/market-data", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, clk, healthResponse{Status: "UP", Components: map[string]healthComponent{
			"market-data": component("market-data", "UP", "Market Data Service (not configured)", nil),
		}})
	})
	mux.HandleFunc("GET /api/health/llm", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, clk, healthResponse{Status: "UP", Components: map[string]healthComponent{
			"llm": component("llm", "UP", "LLM Service (not configured)", nil),
		}})
	})
	mux.HandleFunc("GET /api/health/full", func(w http.ResponseWriter, _ *http.Request) {
		active := func(name, description string) healthComponent {
			return component(name, "UP", description, map[string]any{"status": "Active", "responseTime": "OK"})
		}
		writeHealth(w, clk, healthResponse{Status: "UP", Components: map[string]healthComponent{
			"api":         component("api", "UP", "REST API Service", nil),
			"database":    active("database", "Database Service"),
			"market-data": active("market-data", "Market Data Service"),
			"strategy":    active("strategy", "Trading Strategy Service"),
			"llm":         active("llm", "LLM Sentiment Service"),
		}})
	})
}

func component(name, status, description string, details map[string]any) healthComponent {
	if details == nil {
		details = map[string]any{}
	}
	return healthComponent{Name: name, Status: status, Description: description, Details: details}
}

func dataDescription(configured bool) string {
	if configured {
		return "Data Ingestion Service"
	}
	return "Data Service (not configured)"
}

func dataDetails(configured bool) map[string]any {
	if !configured {
		return nil
	}
	return map[string]any{"serviceName": "WatchlistReader", "status": "Active"}
}

func queryBool(value string) bool {
	return strings.EqualFold(value, "true")
}

func formatUptime(value time.Duration) string {
	minutes := int64(value.Minutes())
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func memoryMegabytes(value uint64) int64 {
	megabytes := value / uint64(memoryMegabyte)
	const maxInt64 = int64(^uint64(0) >> 1)
	if megabytes > uint64(maxInt64) {
		return maxInt64
	}
	return int64(megabytes)
}

func writeHealth(w http.ResponseWriter, clk clock.Clock, response healthResponse) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	response.SystemInfo = healthSystemInfo{
		JavaVersion: runtime.Version(), OSName: runtime.GOOS, OSVersion: "unknown", Architecture: runtime.GOARCH,
		AvailableProcessors: runtime.NumCPU(), FreeMemoryMB: memoryMegabytes(memory.HeapIdle - memory.HeapReleased),
		MaxMemoryMB: memoryMegabytes(memory.HeapSys), TotalMemoryMB: memoryMegabytes(memory.HeapAlloc),
	}
	response.Timestamp = clk.Now().Format("2006-01-02T15:04:05.000000")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
