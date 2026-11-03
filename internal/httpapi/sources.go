package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

func (a *API) sources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.createSource(w, r)
	case http.MethodGet:
		a.listSources(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) createSource(w http.ResponseWriter, r *http.Request) {
	var request createSourceRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	videoID, _, err := youtube.ParseVideoURL(request.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid YouTube URL")
		return
	}
	source, err := a.repository.CreateSource(r.Context(), videoID, request.URL)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	metadata, err := a.metadataProvider.FetchMetadata(r.Context(), videoID)
	if err == nil {
		if err := a.repository.UpdateSourceMetadata(r.Context(), source.ID, metadata); err != nil {
			writeStoreError(w, err)
			return
		}
		source, err = a.repository.GetSource(r.Context(), source.ID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, toSourceResponse(source))
}

func (a *API) listSources(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, err := queryInt(query.Get("limit"), 50, 1, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	offset, err := queryInt(query.Get("offset"), 0, 0, int(^uint(0)>>1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}
	status := query.Get("status")
	videoID, ok := sourceVideoIDFilter(query)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid video_id filter")
		return
	}
	sources, err := a.repository.ListSources(r.Context(), status, videoID, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid source filter or pagination")
		return
	}
	items := make([]sourceResponse, 0, len(sources))
	for _, source := range sources {
		items = append(items, toSourceResponse(source))
	}
	writeJSON(w, http.StatusOK, sourceListResponse{Items: items, Limit: limit, Offset: offset})
}

func sourceVideoIDFilter(query url.Values) (string, bool) {
	values, present := query["video_id"]
	if !present {
		return "", true
	}
	if len(values) != 1 || values[0] == "" {
		return "", false
	}
	videoID, _, err := youtube.ParseVideoURL("https://www.youtube.com/watch?v=" + url.QueryEscape(values[0]))
	if err != nil {
		return "", false
	}
	return videoID, true
}

func (a *API) source(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sources/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid source ID")
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "audio":
			a.serveAudio(w, r, id)
		case "retry":
			a.retrySource(w, r, id)
		default:
			http.NotFound(w, r)
		}
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		source, err := a.repository.GetSource(r.Context(), id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toSourceResponse(source))
	case http.MethodDelete:
		source, err := a.repository.GetSource(r.Context(), id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := a.repository.DeleteSource(r.Context(), id); err != nil {
			writeStoreError(w, err)
			return
		}
		if err := removeAudio(source); err != nil {
			writeError(w, http.StatusInternalServerError, "source deleted but audio cleanup failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func queryInt(raw string, fallback, minimum, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, errors.New("query value is outside its valid range")
	}
	return value, nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "source not found")
	case errors.Is(err, store.ErrDuplicate), errors.Is(err, store.ErrConflict), errors.Is(err, store.ErrNoPending), errors.Is(err, store.ErrPaused):
		writeError(w, http.StatusConflict, "request conflicts with current state")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
