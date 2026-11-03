package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"
)

const metadataLookupTimeout = 30 * time.Second

// FetchMetadata retrieves video details without downloading audio.
func (d *Downloader) FetchMetadata(ctx context.Context, videoID string) (VideoMetadata, error) {
	if ctx == nil {
		return VideoMetadata{}, fmt.Errorf("metadata context is required")
	}
	metadataCtx, cancel := context.WithTimeout(ctx, metadataLookupTimeout)
	defer cancel()
	info, err := d.readMetadata(metadataCtx, videoID)
	if err != nil {
		return VideoMetadata{}, err
	}
	return optionalVideoMetadata(info), nil
}

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
	metadata := optionalVideoMetadata(info)
	metadata.DurationSeconds = int64(math.Ceil(info.Duration))
	return metadata, nil
}

func optionalVideoMetadata(info ytDLPMetadata) VideoMetadata {
	metadata := VideoMetadata{
		Title: info.Title, ChannelName: firstNonEmpty(info.Channel, info.Uploader),
		ChannelID:    firstNonEmpty(info.ChannelID, info.UploaderID),
		ChannelURL:   firstNonEmpty(info.ChannelURL, info.UploaderURL),
		ThumbnailURL: info.Thumbnail, PublishedAt: publicationDate(info.UploadDate, info.ReleaseDate),
		Description: info.Description, Tags: info.Tags, Categories: info.Categories,
		ViewCount: info.ViewCount, Language: info.Language, LiveStatus: info.LiveStatus,
	}
	if info.FileSize != nil && *info.FileSize > 0 {
		metadata.SizeBytes = *info.FileSize
	} else if info.FileSizeApprox != nil && *info.FileSizeApprox > 0 {
		metadata.SizeBytes = *info.FileSizeApprox
	}
	if info.Duration > 0 && !math.IsNaN(info.Duration) && !math.IsInf(info.Duration, 0) {
		metadata.DurationSeconds = int64(math.Ceil(info.Duration))
	}
	return metadata
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func publicationDate(values ...string) string {
	for _, value := range values {
		if parsed, err := time.Parse("20060102", value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return ""
}
