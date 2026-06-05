package server

import (
	_ "embed"
	"net/http"
)

//go:embed dashboard.html
var dashboardHTML string

// DashboardHandler returns an HTTP handler that serves the static dashboard page.
func DashboardHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dashboard" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(dashboardHTML))
	})
}
