package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
)

func TestListSourcesFiltersByVideoID(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	if _, err := queue.CreateSource(ctx, "aaaaaaaaaaa", "https://youtu.be/aaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.CreateSource(ctx, "bbbbbbbbbbb", "https://youtu.be/bbbbbbbbbbb"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(queue, nil, nil, producerTestToken, adminTestToken, sessionTestSecret)
	request := httptest.NewRequest(http.MethodGet, "/sources?video_id=aaaaaaaaaaa", nil)
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /sources by video_id status=%d body=%s", response.Code, response.Body.String())
	}
	var body sourceListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].VideoID != "aaaaaaaaaaa" {
		t.Fatalf("filtered sources = %+v, want only video_id aaaaaaaaaaa", body.Items)
	}
}

func TestListSourcesRejectsInvalidVideoIDFilter(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	api := NewAPI(queue, nil, nil, producerTestToken, adminTestToken, sessionTestSecret)
	request := httptest.NewRequest(http.MethodGet, "/sources?video_id=invalid", nil)
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /sources with invalid video_id status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestListSourcesCombinesVideoIDAndStatusFilters(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	if _, err := queue.CreateSource(ctx, "aaaaaaaaaaa", "https://youtu.be/aaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(queue, nil, nil, producerTestToken, adminTestToken, sessionTestSecret)
	request := httptest.NewRequest(http.MethodGet, "/sources?video_id=aaaaaaaaaaa&status=ready", nil)
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /sources with combined filters status=%d body=%s", response.Code, response.Body.String())
	}
	var body sourceListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("combined filters returned %+v, want no ready sources", body.Items)
	}
}

func TestListSourcesReturnsEmptyForUnknownVideoID(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	api := NewAPI(queue, nil, nil, producerTestToken, adminTestToken, sessionTestSecret)
	request := httptest.NewRequest(http.MethodGet, "/sources?video_id=ccccccccccc", nil)
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /sources with unknown video_id status=%d body=%s", response.Code, response.Body.String())
	}
	var body sourceListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("unknown video_id returned %+v, want empty items", body.Items)
	}
}

