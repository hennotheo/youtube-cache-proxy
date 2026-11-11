package discovery

import (
	"context"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSearchAndDetails(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("key") != "secret" {
			t.Error("missing API key")
		}
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("type") != "video" || r.URL.Query().Get("q") != "test" || r.URL.Query().Get("maxResults") != "50" {
				t.Error(r.URL)
			}
			w.Write([]byte(`{"items":[{"id":{"videoId":"abc123def45"}},{"id":{"videoId":"abc123def45"}}]}`))
		case "/videos":
			w.Write([]byte(`{"items":[{"id":"abc123def45","snippet":{"title":"Title","channelId":"channel","publishedAt":"2026-01-01T00:00:00Z"},"contentDetails":{"duration":"PT3M"}}]}`))
		default:
			t.Error(r.URL)
		}
	}))
	defer srv.Close()
	c := &Client{Key: "secret", BaseURL: srv.URL, HTTP: srv.Client()}
	videos, err := c.Search(context.Background(), store.SearchInput{Query: "test"})
	if err != nil || len(videos) != 1 || videos[0].Duration != "PT3M" || calls != 2 {
		t.Fatalf("%+v %v calls=%d", videos, err, calls)
	}
}
func TestClientErrorsAreSanitized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "secret", 403) }))
	defer srv.Close()
	c := &Client{Key: "secret", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Search(context.Background(), store.SearchInput{Query: "test"})
	if err == nil || err.Error() != "youtube HTTP 403" {
		t.Fatal(err)
	}
}
