package youtube

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadRejectsMetadataOutsideLimitsBeforeTransfer(t *testing.T) {
	for _, test := range []struct {
		name     string
		metadata string
		limits   DownloadLimits
		wantErr  error
	}{
		{
			name:     "duration",
			metadata: `{"title":"test","duration":3601,"filesize":100}`,
			limits:   DownloadLimits{MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: time.Second},
			wantErr:  ErrVideoTooLong,
		},
		{
			name:     "known filesize",
			metadata: `{"title":"test","duration":60,"filesize":1001}`,
			limits:   DownloadLimits{MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: time.Second},
			wantErr:  ErrDownloadTooLarge,
		},
		{
			name:     "unknown duration",
			metadata: `{"title":"test","filesize":100}`,
			limits:   DownloadLimits{MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: time.Second},
			wantErr:  ErrVideoDurationUnknown,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			outputTemplate := testOutputTemplate(t)
			d := newLimitsTestDownloader(t, "wait", test.metadata, outputTemplate, &calls)
			if _, err := d.Download(context.Background(), "abc123def45", outputTemplate, test.limits); !errors.Is(err, test.wantErr) {
				t.Fatalf("Download error = %v, want %v", err, test.wantErr)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("yt-dlp invocations = %d, want metadata lookup only", got)
			}
		})
	}
}

func TestDownloadRejectsOutputTemplateForAnotherVideo(t *testing.T) {
	var calls atomic.Int32
	outputTemplate := filepath.Join(t.TempDir(), "differentID1.%(ext)s")
	d := newLimitsTestDownloader(t, "complete", `{"title":"test","duration":60}`, outputTemplate, &calls)
	_, err := d.Download(context.Background(), "abc123def45", outputTemplate, DownloadLimits{
		MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("output template for a different video was accepted")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("yt-dlp invocations = %d, want none", got)
	}
}

func TestDownloadStopsWhenActualBytesExceedLimitAndRemovesPartial(t *testing.T) {
	outputTemplate := testOutputTemplate(t)
	d := newLimitsTestDownloader(t, "oversize", `{"title":"test","duration":60}`, outputTemplate, nil)
	_, err := d.Download(context.Background(), "abc123def45", outputTemplate, DownloadLimits{
		MaxBytes: 100, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: 3 * time.Second,
	})
	if !errors.Is(err, ErrDownloadTooLarge) {
		t.Fatalf("Download error = %v, want ErrDownloadTooLarge", err)
	}
	assertNoStagingDirectories(t, filepath.Dir(outputTemplate))
}

func TestDownloadTimeoutStopsProcessAndRemovesPartial(t *testing.T) {
	outputTemplate := testOutputTemplate(t)
	d := newLimitsTestDownloader(t, "wait", `{"title":"test","duration":60}`, outputTemplate, nil)
	_, err := d.Download(context.Background(), "abc123def45", outputTemplate, DownloadLimits{
		MaxBytes: 1000, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: 250 * time.Millisecond,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Download error = %v, want context.DeadlineExceeded", err)
	}
	assertNoStagingDirectories(t, filepath.Dir(outputTemplate))
}

func TestDownloadChecksActualOutputSizeAndReturnsMetadata(t *testing.T) {
	outputTemplate := testOutputTemplate(t)
	outputPath := filepath.Join(filepath.Dir(outputTemplate), "abc123def45.webm")
	d := newLimitsTestDownloader(t, "complete", `{"title":"title","duration":60,"filesize":2}`, outputTemplate, nil)
	metadata, err := d.Download(context.Background(), "abc123def45", outputTemplate, DownloadLimits{
		MaxBytes: 10, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "title" || metadata.DurationSeconds != 60 || metadata.SizeBytes != 5 {
		t.Fatalf("metadata = %+v, want title, 60 seconds, and 5 actual bytes", metadata)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("completed output file is missing: %v", err)
	}
}

func TestDownloadRemovesOutputWhenActualSizeIsTooLarge(t *testing.T) {
	outputTemplate := testOutputTemplate(t)
	outputPath := filepath.Join(filepath.Dir(outputTemplate), "abc123def45.webm")
	d := newLimitsTestDownloader(t, "complete", `{"title":"title","duration":60}`, outputTemplate, nil)
	_, err := d.Download(context.Background(), "abc123def45", outputTemplate, DownloadLimits{
		MaxBytes: 4, MaxRateBytesPerSecond: 5000000, MaxDurationSeconds: 3600, Timeout: 3 * time.Second,
	})
	if !errors.Is(err, ErrDownloadTooLarge) {
		t.Fatalf("Download error = %v, want ErrDownloadTooLarge", err)
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("oversized output still exists: stat error = %v", err)
	}
}

func newLimitsTestDownloader(t *testing.T, downloadMode, metadata, outputTemplate string, calls *atomic.Int32) *Downloader {
	t.Helper()
	return &Downloader{
		executable: "fake-yt-dlp",
		commandContext: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			if calls != nil {
				calls.Add(1)
			}
			mode := downloadMode
			if containsArgument(args, "--dump-single-json") {
				mode = "metadata"
			}
			outputTemplate := argumentValue(args, "--output")
			outputPath := strings.TrimSuffix(outputTemplate, outputExtensionTemplate) + "webm"
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloaderLimitsHelperProcess$")
			cmd.Env = append(os.Environ(), "YTDLP_LIMITS_HELPER=1", "YTDLP_LIMITS_MODE="+mode,
				"YTDLP_LIMITS_METADATA="+metadata,
				"YTDLP_LIMITS_PARTIAL="+outputPath+".part",
				"YTDLP_LIMITS_OUTPUT="+outputPath)
			return cmd
		},
	}
}

func argumentValue(args []string, name string) string {
	for index, arg := range args {
		if arg == name && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func assertNoStagingDirectories(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".download-") {
			t.Errorf("staging directory %q remains", entry.Name())
		}
	}
}

func testOutputTemplate(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "abc123def45.%(ext)s")
}

func containsArgument(args []string, expected string) bool {
	for _, arg := range args {
		if arg == expected {
			return true
		}
	}
	return false
}

func TestDownloaderLimitsHelperProcess(t *testing.T) {
	if os.Getenv("YTDLP_LIMITS_HELPER") != "1" {
		return
	}
	switch os.Getenv("YTDLP_LIMITS_MODE") {
	case "metadata":
		_, _ = os.Stdout.WriteString(os.Getenv("YTDLP_LIMITS_METADATA"))
		os.Exit(0)
	case "oversize":
		if err := os.WriteFile(os.Getenv("YTDLP_LIMITS_PARTIAL"), []byte(strings.Repeat("x", 101)), 0600); err != nil {
			t.Fatal(err)
		}
		for {
			time.Sleep(time.Second)
		}
	case "complete":
		if err := os.WriteFile(os.Getenv("YTDLP_LIMITS_OUTPUT"), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
		os.Exit(0)
	default:
		if err := os.WriteFile(os.Getenv("YTDLP_LIMITS_PARTIAL"), []byte("partial"), 0600); err != nil {
			t.Fatal(err)
		}
		for {
			time.Sleep(time.Second)
		}
	}
}
