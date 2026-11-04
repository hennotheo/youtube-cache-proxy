package httpapi

import "net/http"

type errorResponseWriter struct {
	http.ResponseWriter
	failed bool
}

// WriteHeader replaces file-server errors with the API error DTO.
func (w *errorResponseWriter) WriteHeader(status int) {
	if w.failed {
		return
	}
	if status < 400 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.failed = true
	w.Header().Del("Content-Length")
	w.Header().Del("Content-Disposition")
	writeError(w.ResponseWriter, status, audioErrorMessage(status))
}

// Write discards the file server's plain-text error body after converting it.
func (w *errorResponseWriter) Write(body []byte) (int, error) {
	if w.failed {
		return len(body), nil
	}
	return w.ResponseWriter.Write(body)
}

// serveAudioFile adapts standard file serving errors to the API JSON format.
func serveAudioFile(w http.ResponseWriter, r *http.Request, path string) {
	writer := &errorResponseWriter{ResponseWriter: w}
	http.ServeFile(writer, r, path)
}

func audioErrorMessage(status int) string {
	switch status {
	case http.StatusNotFound:
		return "audio file not found"
	case http.StatusRequestedRangeNotSatisfiable:
		return "requested range is not satisfiable"
	default:
		return "internal server error"
	}
}

