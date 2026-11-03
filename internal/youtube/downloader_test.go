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
		_, err := downloader.Download(ctx, "abc123def45", "data/audio/abc123def45.%(ext)s", DownloadLimits{
			MaxBytes: 1000, MaxDurationSeconds: 3600, Timeout: 5 * time.Second,
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

func TestDownloaderHelperProcess(t *testing.T) {
	if os.Getenv("YTDLP_TEST_HELPER") != "1" {
		return
	}
	if os.Getenv("YTDLP_TEST_MODE") == "metadata" {
		_, _ = os.Stdout.WriteString(`{"title":"test","duration":60}`)
		os.Exit(0)
	}
	if err := os.WriteFile(os.Getenv("YTDLP_TEST_READY_FILE"), []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
