package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

func TestStartReservesBatchAndReturnsBeforeDownloadFinishes(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	first, err := queue.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	second, err := queue.CreateSource(ctx, "bbbbbbbbbbb", "https://www.youtube.com/watch?v=bbbbbbbbbbb")
	if err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(t.TempDir(), "audio")
	blockedDownloader := &fakeDownloader{started: make(chan struct{}, 1), release: make(chan struct{})}
	worker, err := New(ctx, queue, blockedDownloader, outputDir, testDownloadLimits())
	if err != nil {
		t.Fatal(err)
	}
	var releaseOnce sync.Once
	releaseDownload := func() { releaseOnce.Do(func() { close(blockedDownloader.release) }) }
	defer releaseDownload()

	startDone := make(chan struct{})
	var run store.DownloadRun
	var startErr error
	go func() {
		run, startErr = worker.Start(ctx, 1)
		close(startDone)
	}()
	select {
	case <-startDone:
	case <-time.After(time.Second):
		t.Fatal("Start waited for the download to finish")
	}
	if startErr != nil {
		t.Fatal(startErr)
	}
	if run.RequestedCount != 1 || run.ReservedCount != 1 || run.Status != "active" {
		t.Fatalf("unexpected batch: %+v", run)
	}
	select {
	case <-blockedDownloader.started:
	case <-time.After(time.Second):
		t.Fatal("background download did not start")
	}
	if _, err := worker.Start(ctx, 1); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("second Start error = %v, want store.ErrConflict", err)
	}

	releaseDownload()
	worker.Wait()

	completed, err := queue.GetSource(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "ready" || completed.AudioPath.String != filepath.Join(outputDir, "aaaaaaaaaaa.webm") || completed.AudioMime.String != "audio/webm" {
		t.Fatalf("first source after batch = %+v, want ready local webm audio", completed)
	}
	untouched, err := queue.GetSource(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if untouched.Status != "pending" {
		t.Fatalf("source outside requested batch status = %q, want pending", untouched.Status)
	}
}

func TestBatchRecordsFailuresAndContinuesWithTheNextReservedSource(t *testing.T) {
	ctx := context.Background()
	queue, err := store.Open(ctx, filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	failedSource, err := queue.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	readySource, err := queue.CreateSource(ctx, "bbbbbbbbbbb", "https://www.youtube.com/watch?v=bbbbbbbbbbb")
	if err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(t.TempDir(), "audio")
	d := &fakeDownloader{failureVideoID: failedSource.VideoID, failure: context.DeadlineExceeded}
	worker, err := New(ctx, queue, d, outputDir, testDownloadLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.Start(ctx, 2); err != nil {
		t.Fatal(err)
	}
	worker.Wait()

	failed, err := queue.GetSource(ctx, failedSource.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != "failed" || failed.ErrorCode.String != "temporary_download_failure" {
		t.Fatalf("failed source = status %q, error code %q; want failed and temporary_download_failure", failed.Status, failed.ErrorCode.String)
	}
	ready, err := queue.GetSource(ctx, readySource.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != "ready" {
		t.Fatalf("source after failed download status = %q, want ready", ready.Status)
	}
}

type fakeDownloader struct {
	started        chan struct{}
	release        chan struct{}
	failureVideoID string
	failure        error
}

func (d *fakeDownloader) Download(ctx context.Context, videoID, outputTemplate string, _ youtube.DownloadLimits) (youtube.VideoMetadata, error) {
	if d.started != nil {
		d.started <- struct{}{}
	}
	if d.release != nil {
		select {
		case <-d.release:
		case <-ctx.Done():
			return youtube.VideoMetadata{}, ctx.Err()
		}
	}
	if videoID == d.failureVideoID && d.failure != nil {
		return youtube.VideoMetadata{}, d.failure
	}
	outputPath := strings.Replace(outputTemplate, "%(ext)s", "webm", 1)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return youtube.VideoMetadata{}, err
	}
	if err := os.WriteFile(outputPath, []byte("audio"), 0600); err != nil {
		return youtube.VideoMetadata{}, err
	}
	return youtube.VideoMetadata{Title: videoID, DurationSeconds: 12, SizeBytes: 5}, nil
}

func testDownloadLimits() youtube.DownloadLimits {
	return youtube.DownloadLimits{
		MaxBytes:              1000,
		MaxRateBytesPerSecond: 5000,
		MaxDurationSeconds:    60,
		Timeout:               time.Second,
	}
}
