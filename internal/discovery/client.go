// Package discovery provides metadata-only YouTube discovery, independent of downloads.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Key, BaseURL string
	HTTP         *http.Client
}

func NewClient(key string) *Client {
	return &Client{Key: key, BaseURL: "https://www.googleapis.com/youtube/v3", HTTP: &http.Client{Timeout: 30 * time.Second}}
}
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	q.Set("key", c.Key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("youtube request invalid")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("youtube unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("youtube HTTP %d", resp.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out); err != nil {
		return fmt.Errorf("youtube response invalid")
	}
	return nil
}
func (c *Client) Search(ctx context.Context, in store.SearchInput) ([]store.DiscoveryVideo, error) {
	q := url.Values{"part": {"snippet"}, "type": {"video"}, "maxResults": {"50"}, "order": {"date"}}
	for k, v := range map[string]string{"q": in.Query, "channelId": in.ChannelID, "publishedAfter": in.PublishedAfter, "publishedBefore": in.PublishedBefore} {
		if v != "" {
			q.Set(k, v)
		}
	}
	var result struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			}
		}
	}
	if err := c.get(ctx, "/search", q, &result); err != nil {
		return nil, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, item := range result.Items {
		if id := item.ID.VideoID; id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	videos := make([]store.DiscoveryVideo, 0, len(ids))
	if len(ids) == 0 {
		return videos, nil
	}
	var details struct {
		Items []struct {
			ID      string
			Snippet struct {
				Title, Description, ChannelID, ChannelTitle, PublishedAt string
				Thumbnails                                               struct {
					Medium  struct{ URL string }
					Default struct{ URL string }
				}
			}
			ContentDetails struct{ Duration string }
		}
	}
	if err := c.get(ctx, "/videos", url.Values{"part": {"snippet,contentDetails"}, "id": {strings.Join(ids, ",")}}, &details); err != nil {
		return nil, err
	}
	for _, item := range details.Items {
		if !seen[item.ID] {
			continue
		}
		thumb := item.Snippet.Thumbnails.Medium.URL
		if thumb == "" {
			thumb = item.Snippet.Thumbnails.Default.URL
		}
		videos = append(videos, store.DiscoveryVideo{VideoID: item.ID, Title: item.Snippet.Title, Description: item.Snippet.Description, ChannelID: item.Snippet.ChannelID, ChannelTitle: item.Snippet.ChannelTitle, PublishedAt: item.Snippet.PublishedAt, Thumbnail: thumb, Duration: item.ContentDetails.Duration})
	}
	return videos, nil
}
