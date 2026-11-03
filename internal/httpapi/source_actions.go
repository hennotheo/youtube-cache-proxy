package httpapi

import (
	"errors"
	"net/http"
	"os"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
)

func (a *API) retrySource(w http.ResponseWriter, r *http.Request, sourceID int64) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := a.repository.RetrySource(r.Context(), sourceID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) runDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request runRequest
	if err := decodeJSON(w, r, &request); err != nil || request.Count < 1 || request.Count > 100 {
		writeError(w, http.StatusBadRequest, "count must be between 1 and 100")
		return
	}
	run, err := a.worker.Start(r.Context(), request.Count)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, downloadRunResponse{
		ID: run.ID, RequestedCount: run.RequestedCount, ReservedCount: run.ReservedCount,
		Status: run.Status, CreatedAt: run.CreatedAt,
	})
}

func removeAudio(source store.Source) error {
	if !source.AudioPath.Valid || source.AudioPath.String == "" {
		return nil
	}
	err := os.Remove(source.AudioPath.String)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
