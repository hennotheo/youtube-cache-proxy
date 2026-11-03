package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
