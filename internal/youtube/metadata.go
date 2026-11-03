package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
)

func (d *Downloader) readMetadata(ctx context.Context, videoID string) (ytDLPMetadata, error) {
	args, err := buildMetadataArgs(videoID)
	if err != nil {
		return ytDLPMetadata{}, err
	}
	cmd := d.commandContext(ctx, d.executable, args...)
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return ytDLPMetadata{}, ctx.Err()
		}
		var stderr string
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderr = string(exitError.Stderr)
		}
		return ytDLPMetadata{}, fmt.Errorf("read yt-dlp metadata: %w", classifyYTDLPFailure(stderr, err))
	}

	var metadata ytDLPMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return ytDLPMetadata{}, fmt.Errorf("decode yt-dlp metadata: %w", err)
	}
	return metadata, nil
}

func videoMetadata(info ytDLPMetadata) (VideoMetadata, error) {
	if info.Duration <= 0 || math.IsNaN(info.Duration) || math.IsInf(info.Duration, 0) {
		return VideoMetadata{}, ErrVideoDurationUnknown
	}
	metadata := VideoMetadata{Title: info.Title, DurationSeconds: int64(math.Ceil(info.Duration))}
	if info.FileSize != nil && *info.FileSize > 0 {
		metadata.SizeBytes = *info.FileSize
	} else if info.FileSizeApprox != nil && *info.FileSizeApprox > 0 {
		metadata.SizeBytes = *info.FileSizeApprox
	}
	return metadata, nil
}
