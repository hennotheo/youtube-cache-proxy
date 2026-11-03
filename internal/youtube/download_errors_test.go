package youtube

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestReadMetadataClassifiesYTDLPFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		stderr    string
		retryable bool
	}{
		{name: "temporary HTTP failure", stderr: "ERROR: HTTP Error 503: Service Unavailable", retryable: true},
		{name: "temporary network failure", stderr: "ERROR: The read operation timed out", retryable: true},
		{name: "unavailable video", stderr: "ERROR: [youtube] abc123def45: Video unavailable", retryable: false},
		{name: "unknown failure defaults to permanent", stderr: "ERROR: unexpected extractor failure", retryable: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			downloader := newErrorTestDownloader(test.stderr)
			_, err := downloader.readMetadata(context.Background(), "abc123def45")
			if err == nil {
				t.Fatal("readMetadata error = nil, want failure")
			}
			if got := IsRetryableDownloadError(err); got != test.retryable {
				t.Fatalf("IsRetryableDownloadError(%v) = %t, want %t", err, got, test.retryable)
			}
		})
	}
}

func TestIsRetryableDownloadErrorClassifiesDeadline(t *testing.T) {
	if !IsRetryableDownloadError(context.DeadlineExceeded) {
		t.Fatal("deadline error was not classified as retryable")
	}
	if IsRetryableDownloadError(errors.New("caller canceled")) {
		t.Fatal("unclassified error was classified as retryable")
	}
}

func TestRunDownloadClassifiesYTDLPFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		stderr    string
		retryable bool
	}{
		{name: "temporary HTTP failure", stderr: "ERROR: HTTP Error 429: Too Many Requests", retryable: true},
		{name: "permanent unavailable video", stderr: "ERROR: This video is private", retryable: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			downloader := newErrorTestDownloader(test.stderr)
			err := downloader.runDownload(context.Background(), "abc123def45", testOutputTemplate(t), 1000, 5000)
			if err == nil {
				t.Fatal("runDownload error = nil, want failure")
			}
			if got := IsRetryableDownloadError(err); got != test.retryable {
				t.Fatalf("IsRetryableDownloadError(%v) = %t, want %t", err, got, test.retryable)
			}
		})
	}
}

func newErrorTestDownloader(stderr string) *Downloader {
	return &Downloader{
		executable: "fake-yt-dlp",
		commandContext: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloaderErrorHelperProcess$")
			cmd.Env = append(os.Environ(), "YTDLP_ERROR_HELPER=1", "YTDLP_ERROR_STDERR="+stderr)
			return cmd
		},
	}
}

func TestDownloaderErrorHelperProcess(t *testing.T) {
	if os.Getenv("YTDLP_ERROR_HELPER") != "1" {
		return
	}
	_, _ = os.Stderr.WriteString(os.Getenv("YTDLP_ERROR_STDERR"))
	os.Exit(1)
}
