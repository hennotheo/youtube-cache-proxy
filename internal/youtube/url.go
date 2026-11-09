package youtube

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// ErrInvalidVideoURL identifies a URL that does not point to one supported YouTube video.
	ErrInvalidVideoURL = errors.New("invalid YouTube video URL")
	videoIDPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
)

// ParseVideoURL extracts a supported YouTube video ID and returns its canonical watch URL.
func ParseVideoURL(raw string) (string, string, error) {
	parsed, err := parseURL(raw)
	if err != nil {
		return "", "", err
	}

	videoID, err := extractVideoID(parsed)
	if err != nil {
		return "", "", err
	}

	return videoID, "https://www.youtube.com/watch?v=" + videoID, nil
}

func parseURL(raw string) (*url.URL, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.Contains(raw, "#") {
		return nil, fmt.Errorf("%w: malformed URL", ErrInvalidVideoURL)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Opaque != "" || parsed.User != nil || parsed.Host == "" || parsed.Port() != "" || strings.Contains(parsed.Host, ":") || parsed.RawPath != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: malformed or non-HTTPS URL", ErrInvalidVideoURL)
	}

	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be":
	default:
		return nil, fmt.Errorf("%w: unsupported host", ErrInvalidVideoURL)
	}

	return parsed, nil
}

func extractVideoID(parsed *url.URL) (string, error) {
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return "", fmt.Errorf("%w: malformed query", ErrInvalidVideoURL)
	}
	if err := validateQuery(query); err != nil {
		return "", err
	}
	if parsed.Path != "/watch" && len(query["v"]) > 0 {
		return "", fmt.Errorf("%w: video ID parameter is unsupported for this path", ErrInvalidVideoURL)
	}

	var videoID string
	switch host, path := strings.ToLower(parsed.Hostname()), parsed.Path; {
	case host == "youtu.be":
		if strings.Count(path, "/") != 1 {
			return "", fmt.Errorf("%w: unsupported short URL path", ErrInvalidVideoURL)
		}
		videoID = strings.TrimPrefix(path, "/")
	case path == "/watch":
		values := query["v"]
		if len(values) != 1 {
			return "", fmt.Errorf("%w: exactly one video ID is required", ErrInvalidVideoURL)
		}
		videoID = values[0]
	case host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" || host == "music.youtube.com":
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) != 2 || (parts[0] != "shorts" && parts[0] != "embed" && parts[0] != "live") {
			return "", fmt.Errorf("%w: unsupported video URL path", ErrInvalidVideoURL)
		}
		videoID = parts[1]
	default:
		return "", fmt.Errorf("%w: unsupported video URL path", ErrInvalidVideoURL)
	}

	if !videoIDPattern.MatchString(videoID) {
		return "", fmt.Errorf("%w: invalid video ID", ErrInvalidVideoURL)
	}
	return videoID, nil
}

func validateQuery(query url.Values) error {
	for key, values := range query {
		if key != "v" && key != "t" && key != "start" && key != "si" && key != "feature" && key != "list" && key != "index" {
			return fmt.Errorf("%w: unsupported query parameter", ErrInvalidVideoURL)
		}
		if len(values) != 1 || values[0] == "" {
			return fmt.Errorf("%w: repeated or empty query parameter", ErrInvalidVideoURL)
		}
	}
	return nil
}
