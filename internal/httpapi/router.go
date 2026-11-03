package httpapi

import "net/http"

// NewRouter creates the HTTP routes exposed by the service.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	return mux
}
