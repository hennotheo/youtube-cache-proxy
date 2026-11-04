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
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	file, err := dashboardFiles.ReadFile("dashboard.html")
	if err != nil {
		http.Error(w, "dashboard unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(file)
}

