package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

const producerTestToken = "producer-token-for-httpapi-tests-123456"
const adminTestToken = "admin-token-for-httpapi-tests-1234567890"

func TestCreateSourceReturnsAndPersistsMetadataWithoutStartingDownload(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })

	viewCount := int64(8765)
	metadata := youtube.VideoMetadata{
		Title: "Harbor live", DurationSeconds: 63, ChannelName: "Våyagere", ChannelID: "@Voyagere",
		ChannelURL: "https://www.youtube.com/@Voyagere", ThumbnailURL: "https://i.ytimg.com/vi/abc123def45/hqdefault.jpg",
		PublishedAt: "2024-05-09", Description: "Un concert à Montréal", Tags: []string{"city", "café"},
		Categories: []string{"Music"}, ViewCount: &viewCount, Language: "fr", LiveStatus: "not_live",
	}
	metadataCalls := 0
	provider := metadataProviderFunc(func(_ context.Context, videoID string) (youtube.VideoMetadata, error) {
		metadataCalls++
		if videoID != "abc123def45" {
			t.Errorf("metadata lookup video ID = %q, want validated ID", videoID)
		}
		return metadata, nil
	})
	api := NewAPI(queue, nil, provider, producerTestToken, adminTestToken)
	request := httptest.NewRequest(http.MethodPost, "/sources", strings.NewReader(`{"url":"https://youtu.be/abc123def45"}`))
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("POST /sources status = %d, body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "pending" || body["channel_name"] != "Våyagere" || body["thumbnail_url"] != metadata.ThumbnailURL || body["published_at"] != "2024-05-09" {
		t.Fatalf("created source response omitted metadata or changed status: %#v", body)
	}
	if body["audio_url"] != nil || body["description"] != metadata.Description {
		t.Fatalf("created response has unexpected audio URL or missing description: %#v", body)
	}
	if tags, ok := body["tags"].([]any); !ok || len(tags) != 2 || tags[1] != "café" {
		t.Fatalf("tags = %#v, want JSON array including Unicode tag", body["tags"])
	}
	if body["view_count"] != float64(8765) {
		t.Fatalf("view count = %#v, want 8765", body["view_count"])
	}

	for _, route := range []string{"/sources", "/sources/1"} {
		request := httptest.NewRequest(http.MethodGet, route, nil)
		request.Header.Set("Authorization", "Bearer "+producerTestToken)
		response := httptest.NewRecorder()
		NewRouter(api).ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"channel_name":"Våyagere"`) {
			t.Fatalf("GET %s status=%d body=%s", route, response.Code, response.Body.String())
		}
	}
	if metadataCalls != 1 {
		t.Fatalf("metadata lookups = %d, want one during source creation", metadataCalls)
	}
}

func TestCreateSourceSucceedsWhenMetadataLookupFails(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	provider := metadataProviderFunc(func(context.Context, string) (youtube.VideoMetadata, error) {
		return youtube.VideoMetadata{}, errors.New("metadata unavailable")
	})
	api := NewAPI(queue, nil, provider, producerTestToken, adminTestToken)
	request := httptest.NewRequest(http.MethodPost, "/sources", strings.NewReader(`{"url":"https://youtu.be/abc123def45"}`))
	request.Header.Set("Authorization", "Bearer "+producerTestToken)
	response := httptest.NewRecorder()
	NewRouter(api).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("metadata failure blocked source creation: status=%d body=%s", response.Code, response.Body.String())
	}
	sources, err := queue.ListSources(ctx, "pending", "", 10, 0)
	if err != nil || len(sources) != 1 || sources[0].Title.Valid || sources[0].Status != "pending" {
		t.Fatalf("source after metadata failure = %+v, err=%v", sources, err)
	}
}

type metadataProviderFunc func(context.Context, string) (youtube.VideoMetadata, error)

func (f metadataProviderFunc) FetchMetadata(ctx context.Context, videoID string) (youtube.VideoMetadata, error) {
	return f(ctx, videoID)
}
