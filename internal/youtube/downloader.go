package youtube

import (
	"context"
	"fmt"
	"io"
	"os/exec"
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

// Download runs yt-dlp for one video ID and writes to the supplied output template.
func (d *Downloader) Download(ctx context.Context, videoID, outputTemplate string) error {
	if ctx == nil {
		return fmt.Errorf("download context is required")
	}
	args, err := buildDownloadArgs(videoID, outputTemplate)
	if err != nil {
		return err
	}
	cmd := d.commandContext(ctx, d.executable, args...)
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("run yt-dlp: %w", err)
	}
	return nil
}
