package youtube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type commandContextFunc func(context.Context, string, ...string) *exec.Cmd

// Downloader runs yt-dlp without a shell using a canonical video URL.
type Downloader struct {
	executable     string
	commandContext commandContextFunc
}

// NewDownloader creates a downloader that finds yt-dlp through PATH.
func NewDownloader() *Downloader {
	return &Downloader{executable: "yt-dlp", commandContext: exec.CommandContext}
}

// Download checks metadata and downloads one video within the supplied limits.
func (d *Downloader) Download(ctx context.Context, videoID, outputTemplate string, limits DownloadLimits) (VideoMetadata, error) {
	if ctx == nil {
		return VideoMetadata{}, fmt.Errorf("download context is required")
	}
	if err := limits.validate(); err != nil {
		return VideoMetadata{}, err
	}
	if err := validateOutputTemplate(outputTemplate, videoID); err != nil {
		return VideoMetadata{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()

	info, err := d.readMetadata(operationCtx, videoID)
	if err != nil {
		return VideoMetadata{}, err
	}
	metadata, err := videoMetadata(info)
	if err != nil {
		return VideoMetadata{}, errors.Join(err, cleanupPartialFiles(outputTemplate))
	}
	if metadata.DurationSeconds > limits.MaxDurationSeconds {
		return VideoMetadata{}, errors.Join(ErrVideoTooLong, cleanupPartialFiles(outputTemplate))
	}
	if metadata.SizeBytes > limits.MaxBytes {
		return VideoMetadata{}, errors.Join(ErrDownloadTooLarge, cleanupPartialFiles(outputTemplate))
	}

	stagingDir, err := os.MkdirTemp(filepath.Dir(outputTemplate), ".download-"+videoID+"-")
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)
	stagingTemplate := filepath.Join(stagingDir, videoID+"."+outputExtensionTemplate)
	if err := d.runDownload(operationCtx, videoID, stagingTemplate, limits.MaxBytes, limits.MaxRateBytesPerSecond); err != nil {
		return VideoMetadata{}, err
	}
	metadata.SizeBytes, err = actualOutputSize(stagingTemplate)
	if err != nil {
		return VideoMetadata{}, errors.Join(err, cleanupPartialFiles(stagingTemplate))
	}
	if metadata.SizeBytes > limits.MaxBytes {
		return VideoMetadata{}, errors.Join(ErrDownloadTooLarge, removeOutputFiles(stagingTemplate))
	}
	if err := publishOutput(stagingTemplate, outputTemplate); err != nil {
		return VideoMetadata{}, err
	}
	return metadata, nil
}

func (d *Downloader) runDownload(ctx context.Context, videoID, outputTemplate string, maxBytes, maxRateBytesPerSecond int64) error {
	args, err := buildDownloadArgs(videoID, outputTemplate, maxBytes, maxRateBytesPerSecond)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := d.commandContext(runCtx, d.executable, args...)
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if ctx.Err() != nil {
			return errors.Join(ctx.Err(), cleanupPartialFiles(outputTemplate))
		}
		return errors.Join(fmt.Errorf("start yt-dlp: %w", err), cleanupPartialFiles(outputTemplate))
	}
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-finished:
			if err != nil {
				if ctx.Err() != nil {
					return errors.Join(ctx.Err(), cleanupPartialFiles(outputTemplate))
				}
				classified := classifyYTDLPFailure(stderr.String(), err)
				return errors.Join(fmt.Errorf("run yt-dlp: %w", classified), cleanupPartialFiles(outputTemplate))
			}
			return nil
		case <-ticker.C:
			size, err := partialOutputSize(outputTemplate)
			if err != nil {
				cancel()
				<-finished
				return errors.Join(fmt.Errorf("check yt-dlp partial file: %w", err), cleanupPartialFiles(outputTemplate))
			}
			if size > maxBytes {
				cancel()
				<-finished
				return errors.Join(ErrDownloadTooLarge, cleanupPartialFiles(outputTemplate))
			}
		case <-ctx.Done():
			cancel()
			<-finished
			return errors.Join(ctx.Err(), cleanupPartialFiles(outputTemplate))
		}
	}
}
