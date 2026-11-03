package youtube

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloaderPassesArgumentsAndStopsProcessWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readyPath := filepath.Join(t.TempDir(), "ready")
	outputTemplate := testOutputTemplate(t)

	type commandCall struct {
		name string
		args []string
	}
	invocations := make(chan commandCall, 1)
	downloader := &Downloader{
		executable: "yt-dlp",
		commandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			invocations <- commandCall{name: name, args: append([]string(nil), args...)}
			mode := "download"
			if containsArgument(args, "--dump-single-json") {
				mode = "metadata"
			}
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloaderHelperProcess$")
			cmd.Env = append(os.Environ(), "YTDLP_TEST_HELPER=1", "YTDLP_TEST_MODE="+mode, "YTDLP_TEST_READY_FILE="+readyPath)
			return cmd
		},
	}
	done := make(chan error, 1)
	go func() {
		_, err := downloader.Download(ctx, "abc123def45", outputTemplate, DownloadLimits{
			MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: 5 * time.Second,
		})
		done <- err
	}()

	call := <-invocations
	if call.name != "yt-dlp" {
		t.Fatalf("executable = %q, want yt-dlp", call.name)
	}
	if want := "https://www.youtube.com/watch?v=abc123def45"; call.args[len(call.args)-1] != want {
		t.Fatalf("last argument = %q, want canonical URL %q", call.args[len(call.args)-1], want)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper process did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Download error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Download did not stop after context cancellation")
	}
}

func TestFetchMetadataParsesVideoDetailsWithoutDownloadingAudio(t *testing.T) {
	var args []string
	var executable string
	downloader := &Downloader{
		executable: "yt-dlp",
		commandContext: func(ctx context.Context, name string, callArgs ...string) *exec.Cmd {
			executable = name
			args = append([]string(nil), callArgs...)
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloaderHelperProcess$")
			cmd.Env = append(os.Environ(), "YTDLP_TEST_HELPER=1", "YTDLP_TEST_MODE=metadata_extended")
			return cmd
		},
	}

	metadata, err := downloader.FetchMetadata(context.Background(), "abc123def45")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Harbor live" || metadata.DurationSeconds != 63 {
		t.Fatalf("basic metadata = %+v", metadata)
	}
	if metadata.ChannelName != "Våyagere" || metadata.ChannelID != "@Voyagere" || metadata.ChannelURL != "https://www.youtube.com/@Voyagere" {
		t.Fatalf("channel metadata = %+v", metadata)
	}
	if metadata.ThumbnailURL != "https://i.ytimg.com/vi/abc123def45/hqdefault.jpg" || metadata.PublishedAt != "2024-05-09" {
		t.Fatalf("thumbnail/publication metadata = %+v", metadata)
	}
	if metadata.Description != "Un concert à Montréal" || len(metadata.Tags) != 2 || metadata.Tags[1] != "café" || len(metadata.Categories) != 1 || metadata.Categories[0] != "Music" {
		t.Fatalf("text metadata = %+v", metadata)
	}
	if metadata.ViewCount == nil || *metadata.ViewCount != 123456 || metadata.Language != "fr" || metadata.LiveStatus != "not_live" {
		t.Fatalf("additional metadata = %+v", metadata)
	}
	if !containsArgument(args, "--skip-download") || containsArgument(args, "--extract-audio") || containsArgument(args, "--output") {
		t.Fatalf("metadata arguments started an audio download: %v", args)
	}
	if executable != "yt-dlp" || !containsArgument(args, "--no-cookies") || !containsArgument(args, "--no-cookies-from-browser") {
		t.Fatalf("metadata lookup executable or cookie arguments are unsafe: executable=%q args=%v", executable, args)
	}
	if got, want := args[len(args)-1], "https://www.youtube.com/watch?v=abc123def45"; got != want {
		t.Fatalf("last argument = %q, want canonical URL %q", got, want)
	}
}

func TestFetchMetadataAllowsMissingOptionalFieldsAndDuration(t *testing.T) {
	downloader := &Downloader{
		executable: "yt-dlp",
		commandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloaderHelperProcess$")
			cmd.Env = append(os.Environ(), "YTDLP_TEST_HELPER=1", "YTDLP_TEST_MODE=metadata_minimal")
			return cmd
		},
	}

	metadata, err := downloader.FetchMetadata(context.Background(), "abc123def45")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Untitled" || metadata.DurationSeconds != 0 || metadata.ChannelName != "" || metadata.Tags != nil || metadata.ViewCount != nil {
		t.Fatalf("missing optional fields were not left empty: %+v", metadata)
	}
}

func TestDownloaderHelperProcess(t *testing.T) {
	if os.Getenv("YTDLP_TEST_HELPER") != "1" {
		return
	}
	if os.Getenv("YTDLP_TEST_MODE") == "metadata" {
		_, _ = os.Stdout.WriteString(`{"title":"test","duration":60}`)
		os.Exit(0)
	}
	if os.Getenv("YTDLP_TEST_MODE") == "metadata_extended" {
		_, _ = os.Stdout.WriteString(`{"title":"Harbor live","duration":62.25,"uploader":"Våyagere","uploader_id":"@Voyagere","uploader_url":"https://www.youtube.com/@Voyagere","thumbnail":"https://i.ytimg.com/vi/abc123def45/hqdefault.jpg","upload_date":"20240509","description":"Un concert à Montréal","tags":["city","café"],"categories":["Music"],"view_count":123456,"language":"fr","live_status":"not_live"}`)
		os.Exit(0)
	}
	if os.Getenv("YTDLP_TEST_MODE") == "metadata_minimal" {
		_, _ = os.Stdout.WriteString(`{"title":"Untitled"}`)
		os.Exit(0)
	}
	if err := os.WriteFile(os.Getenv("YTDLP_TEST_READY_FILE"), []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
