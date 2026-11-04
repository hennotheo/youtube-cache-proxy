package httpapi

import "net/http"

// NewRouter creates the HTTP routes exposed by the service.
func NewRouter(apis ...*API) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", dashboard)
	mux.HandleFunc("/health", health)
	if len(apis) > 0 && apis[0] != nil {
		apis[0].register(mux)
	}
	return mux
}

