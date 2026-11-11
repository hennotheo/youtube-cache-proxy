package httpapi

import (
	"context"
	"github.com/hennotheo/youtube-cache-proxy/internal/discovery"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type discoveryWake struct{ calls int }

func (w *discoveryWake) Notify() { w.calls++ }
func TestDiscoveryAuthAndCache(t *testing.T) {
	s, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wake := &discoveryWake{}
	a := NewAPI(s, nil, nil, "producer", "admin", "session")
	a.EnableDiscovery(s, wake)
	router := NewRouter(a)
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if w := request("POST", "/discovery/searches", `{"query":"test"}`, "producer"); w.Code != http.StatusForbidden {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("POST", "/discovery/searches", `{"query":"test"}`, "admin"); w.Code != 202 {
		t.Fatal(w.Code, w.Body)
	}
	if wake.calls != 1 {
		t.Fatal(wake.calls)
	}
	for _, path := range []string{"/discovery/searches/1", "/discovery/videos?limit=1", "/discovery/videos?limit=1&offset=1"} {
		if w := request("GET", path, "", "producer"); w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if wake.calls != 1 {
		t.Fatal("reads triggered execution")
	}
	for _, body := range []string{`{}`, `{"query":"test","publishedAfter":"wrong"}`, `{"query":"test","publishedAfter":"2026-02-01T00:00:00Z","publishedBefore":"2026-01-01T00:00:00Z"}`} {
		if w := request("POST", "/discovery/searches", body, "admin"); w.Code != 400 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if w := request("GET", "/discovery/videos?sort=unsafe", "", "producer"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := request("GET", "/discovery/videos", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
}

type controlledDiscovery struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (p *controlledDiscovery) Search(ctx context.Context, in store.SearchInput) ([]store.DiscoveryVideo, error) {
	p.calls.Add(1)
	close(p.started)
	select {
	case <-p.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return []store.DiscoveryVideo{{VideoID: "abc123def45", Title: "cached", PublishedAt: "2026-01-01T00:00:00Z"}}, nil
}
func TestDiscoveryAsyncExecutionThenCacheOnlyReads(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	provider := &controlledDiscovery{started: make(chan struct{}), release: make(chan struct{})}
	worker, err := discovery.NewWorker(ctx, s, provider)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	a := NewAPI(s, nil, nil, "producer", "admin", "session")
	a.EnableDiscovery(s, worker)
	router := NewRouter(a)
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		r := httptest.NewRequest("POST", "/discovery/searches", strings.NewReader(`{"query":"test"}`))
		r.Header.Set("Authorization", "Bearer admin")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		finished <- w
	}()
	select {
	case w := <-finished:
		if w.Code != 202 {
			t.Fatal(w.Code, w.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP waited for YouTube")
	}
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not start")
	}
	// SQLite reads remain available while the provider is blocked.
	r := httptest.NewRequest("GET", "/discovery/videos", nil)
	r.Header.Set("Authorization", "Bearer producer")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	close(provider.release)
	deadline := time.Now().Add(3 * time.Second)
	for {
		q, e := s.GetSearch(ctx, 1)
		if e != nil {
			t.Fatal(e)
		}
		if q.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(q)
		}
		time.Sleep(time.Millisecond * 10)
	}
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "cached") {
			t.Fatal(w.Code, w.Body)
		}
	}
	if provider.calls.Load() != 1 {
		t.Fatal("cache reads invoked YouTube", provider.calls.Load())
	}
}
