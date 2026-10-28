package api

import "net/http"

func HealthExtra(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
