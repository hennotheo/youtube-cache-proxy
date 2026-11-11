package httpapi

import (
	"context"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DiscoveryRepository interface {
	CreateSearch(context.Context, store.SearchInput) (store.Search, error)
	GetSearch(context.Context, int64) (store.Search, error)
	ListDiscoveries(context.Context, store.DiscoveryFilter) ([]store.DiscoveryVideo, error)
}
type DiscoveryNotifier interface{ Notify() }

// EnableDiscovery attaches the persistent discovery cache; a nil notifier disables execution.
func (a *API) EnableDiscovery(s DiscoveryRepository, w DiscoveryNotifier) {
	a.discovery = s
	a.discoveryWorker = w
}
func (a *API) registerDiscovery(mux *http.ServeMux) {
	if a.discovery == nil {
		return
	}
	mux.Handle("POST /discovery/searches", a.requireRole(adminRole, a.createDiscoverySearch))
	mux.Handle("GET /discovery/searches/{id}", a.requireRole(producerRole, a.getDiscoverySearch))
	mux.Handle("GET /discovery/videos", a.requireRole(producerRole, a.listDiscoveryVideos))
}
func validateDiscoveryDates(after, before string) bool {
	var a, b time.Time
	var err error
	if after != "" {
		a, err = time.Parse(time.RFC3339, after)
		if err != nil {
			return false
		}
	}
	if before != "" {
		b, err = time.Parse(time.RFC3339, before)
		if err != nil {
			return false
		}
	}
	return a.IsZero() || b.IsZero() || a.Before(b)
}
func (a *API) createDiscoverySearch(w http.ResponseWriter, r *http.Request) {
	var in store.SearchInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeDecodeError(w, err, "invalid JSON request")
		return
	}
	in.Query = strings.TrimSpace(in.Query)
	in.ChannelID = strings.TrimSpace(in.ChannelID)
	if (in.Query == "" && in.ChannelID == "") || len(in.Query) > 500 || len(in.ChannelID) > 128 || !validateDiscoveryDates(in.PublishedAfter, in.PublishedBefore) {
		writeError(w, 400, "invalid discovery search")
		return
	}
	if a.discoveryWorker == nil {
		writeError(w, 503, "discovery execution unavailable")
		return
	}
	job, err := a.discovery.CreateSearch(r.Context(), in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	a.discoveryWorker.Notify()
	w.Header().Set("Location", "/discovery/searches/"+itoa(job.ID))
	writeJSON(w, 202, job)
}
func (a *API) getDiscoverySearch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "invalid search ID")
		return
	}
	job, err := a.discovery.GetSearch(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, job)
}
func (a *API) listDiscoveryVideos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for k, values := range q {
		if len(values) != 1 {
			writeError(w, 400, "duplicate filter")
			return
		}
		switch k {
		case "searchId", "channelId", "publishedAfter", "publishedBefore", "limit", "offset", "sort":
		default:
			writeError(w, 400, "unsupported discovery filter")
			return
		}
	}
	limit, err := queryInt(q.Get("limit"), 50, 1, 100)
	if err != nil {
		writeError(w, 400, "invalid limit")
		return
	}
	offset, err := queryInt(q.Get("offset"), 0, 0, int(^uint(0)>>1))
	if err != nil {
		writeError(w, 400, "invalid offset")
		return
	}
	var searchID int64
	if raw := q.Get("searchId"); raw != "" {
		searchID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || searchID < 1 {
			writeError(w, 400, "invalid searchId")
			return
		}
	}
	sort := q.Get("sort")
	if sort != "" && sort != "publishedAt" && sort != "-publishedAt" {
		writeError(w, 400, "invalid sort")
		return
	}
	if !validateDiscoveryDates(q.Get("publishedAfter"), q.Get("publishedBefore")) {
		writeError(w, 400, "invalid date range")
		return
	}
	items, err := a.discovery.ListDiscoveries(r.Context(), store.DiscoveryFilter{SearchID: searchID, ChannelID: q.Get("channelId"), PublishedAfter: q.Get("publishedAfter"), PublishedBefore: q.Get("publishedBefore"), Limit: limit, Offset: offset, Sort: sort})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, struct {
		Items  []store.DiscoveryVideo `json:"items"`
		Limit  int                    `json:"limit"`
		Offset int                    `json:"offset"`
	}{items, limit, offset})
}
