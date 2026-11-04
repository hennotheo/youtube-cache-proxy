package httpapi

import (
	"embed"
	"net/http"
)

//go:embed dashboard.html
var dashboardFiles embed.FS

// dashboard serves the lightweight browser interface.
func dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	file, err := dashboardFiles.ReadFile("dashboard.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	_, _ = w.Write(file)
}

