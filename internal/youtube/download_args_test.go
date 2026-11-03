package youtube

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildDownloadArgsUsesOnlyCanonicalVideoURL(t *testing.T) {
	outputTemplate := "data/audio folder/abc123def45.%(ext)s"
	args, err := buildDownloadArgs("abc123def45", outputTemplate, 1000, 5_000_000)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--format", "bestaudio/best",
		"--extract-audio",
		"--audio-format", "best",
		"--limit-rate", "5000000",
		"--max-filesize", "1000",
		"--no-progress",
		"--part",
		"--no-overwrites",
		"--output", outputTemplate,
		"https://www.youtube.com/watch?v=abc123def45",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("download args = %#v, want %#v", args, want)
	}
}

func TestBuildDownloadArgsRejectsInvalidVideoID(t *testing.T) {
	if _, err := buildDownloadArgs("https://evil.example/video", "audio.%(ext)s", 1000, 5_000_000); !errors.Is(err, ErrInvalidVideoURL) {
		t.Fatalf("invalid video ID error = %v, want ErrInvalidVideoURL", err)
	}
	if _, err := buildDownloadArgs("abc123def45", "audio.%(ext)s", 0, 5_000_000); err == nil {
		t.Fatal("empty output template was accepted")
	}
}

func TestBuildMetadataArgsUsesOnlyCanonicalVideoURL(t *testing.T) {
	args, err := buildMetadataArgs("abc123def45")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--format", "bestaudio/best",
		"--dump-single-json",
		"--skip-download",
		"https://www.youtube.com/watch?v=abc123def45",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("metadata args = %#v, want %#v", args, want)
	}
}
