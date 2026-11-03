package httpapi

import (
	"net/http"
	"path/filepath"
)

func (a *API) serveAudio(w http.ResponseWriter, r *http.Request, sourceID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source, err := a.repository.GetSource(r.Context(), sourceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if source.Status != "ready" || !source.AudioPath.Valid {
		writeError(w, http.StatusConflict, "audio is not ready")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(source.AudioPath.String)+`"`)
	if source.AudioMime.Valid {
		w.Header().Set("Content-Type", source.AudioMime.String)
	}
	http.ServeFile(w, r, source.AudioPath.String)
}
