package middleware

import (
	"crypto/subtle"
	"net/http"
)

// AdminAPIKey enforces the configured key on admin routes when enabled.
func AdminAPIKey(enabled bool, configured string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || !isAdminPath(r.URL.Path) || !enabled {
				next.ServeHTTP(w, r)
				return
			}
			supplied := r.Header.Get("X-API-Key")
			if configured != "" && subtle.ConstantTimeCompare([]byte(configured), []byte(supplied)) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("WWW-Authenticate", "X-API-Key")
			w.WriteHeader(http.StatusUnauthorized)
		})
	}
}
