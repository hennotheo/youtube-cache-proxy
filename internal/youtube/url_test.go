package youtube

import "testing"

func TestParseVideoURLNormalizesSupportedVideoURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
		id   string
	}{
		{name: "watch", url: "https://www.youtube.com/watch?v=abc123def45", id: "abc123def45"},
		{name: "watch with playlist context", url: "https://www.youtube.com/watch?v=TuabL5wmuw0&list=PLDVJSd5CkHLI&index=3", id: "TuabL5wmuw0"},
		{name: "short link with tracking", url: "https://youtu.be/abc123def45?si=tracking&t=30", id: "abc123def45"},
		{name: "shorts", url: "https://m.youtube.com/shorts/abc123def45", id: "abc123def45"},
		{name: "embed", url: "https://youtube.com/embed/abc123def45", id: "abc123def45"},
		{name: "live", url: "https://music.youtube.com/live/abc123def45", id: "abc123def45"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, normalized, err := ParseVideoURL(test.url)
			if err != nil {
				t.Fatalf("ParseVideoURL(%q) error = %v", test.url, err)
			}
			if id != test.id {
				t.Errorf("video ID = %q, want %q", id, test.id)
			}
			if want := "https://www.youtube.com/watch?v=" + test.id; normalized != want {
				t.Errorf("normalized URL = %q, want %q", normalized, want)
			}
		})
	}
}

func TestParseVideoURLRejectsAmbiguousOrUntrustedURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "empty", url: ""},
		{name: "non HTTPS", url: "http://www.youtube.com/watch?v=abc123def45"},
		{name: "deceptive domain", url: "https://youtube.com.evil.example/watch?v=abc123def45"},
		{name: "credentials", url: "https://youtube.com@evil.example/watch?v=abc123def45"},
		{name: "custom port", url: "https://youtube.com:8443/watch?v=abc123def45"},
		{name: "unknown parameter", url: "https://www.youtube.com/watch?v=abc123def45&redirect=https://evil.example"},
		{name: "repeated timestamp", url: "https://youtu.be/abc123def45?t=1&t=2"},
		{name: "duplicate video ID", url: "https://www.youtube.com/watch?v=abc123def45&v=zyx987wvu65"},
		{name: "playlist path", url: "https://www.youtube.com/playlist?list=PL123"},
		{name: "extra path", url: "https://youtu.be/abc123def45/extra"},
		{name: "invalid video ID", url: "https://www.youtube.com/watch?v=short"},
		{name: "fragment", url: "https://www.youtube.com/watch?v=abc123def45#t=30"},
		{name: "encoded path", url: "https://youtu.be/abc123def45%2Fextra"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := ParseVideoURL(test.url); err == nil {
				t.Errorf("ParseVideoURL(%q) succeeded, want error", test.url)
			}
		})
	}
}
