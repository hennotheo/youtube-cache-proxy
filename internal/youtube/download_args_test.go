package youtube

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildDownloadArgsUsesOnlyCanonicalVideoURL(t *testing.T) {
	outputTemplate := "data/audio folder/abc123def45.%(ext)s"
	args, err := buildDownloadArgs("abc123def45", outputTemplate)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--no-progress",
		"--output", outputTemplate,
		"https://www.youtube.com/watch?v=abc123def45",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("download args = %#v, want %#v", args, want)
	}
}

func TestBuildDownloadArgsRejectsInvalidVideoID(t *testing.T) {
	if _, err := buildDownloadArgs("https://evil.example/video", "audio.%(ext)s"); !errors.Is(err, ErrInvalidVideoURL) {
		t.Fatalf("invalid video ID error = %v, want ErrInvalidVideoURL", err)
	}
	if _, err := buildDownloadArgs("abc123def45", ""); err == nil {
		t.Fatal("empty output template was accepted")
	}
}
