package api

import "net/http"

// HealthExtra is a no-content placeholder for the legacy health probe.
func HealthExtra(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
