package worker

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// audioResult finds the published local file and builds its persisted result.
func audioResult(directory, videoID string, metadata youtube.VideoMetadata) (store.AudioResult, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return store.AudioResult{}, fmt.Errorf("read downloaded audio directory: %w", err)
	}
	prefix := videoID + "."
	var audioPath string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || strings.HasSuffix(entry.Name(), ".part") {
			continue
		}
		if audioPath != "" {
			return store.AudioResult{}, fmt.Errorf("multiple audio outputs found for %s", videoID)
		}
		audioPath = filepath.Join(directory, entry.Name())
	}
	if audioPath == "" {
		return store.AudioResult{}, fmt.Errorf("audio output not found for %s", videoID)
	}
	info, err := os.Stat(audioPath)
	if err != nil {
		return store.AudioResult{}, fmt.Errorf("stat downloaded audio: %w", err)
	}
	return store.AudioResult{
		Title: metadata.Title, DurationSeconds: metadata.DurationSeconds,
		Path: audioPath, Mime: audioContentType(filepath.Ext(audioPath)), SizeBytes: info.Size(), Metadata: metadata,
	}, nil
}

// audioContentType returns an audio MIME type for a supported output extension.
func audioContentType(extension string) string {
	switch strings.ToLower(extension) {
	case ".webm":
		return "audio/webm"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".opus":
		return "audio/opus"
	case ".oga", ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".wav":
		return "audio/wav"
	default:
		if contentType := mime.TypeByExtension(extension); strings.HasPrefix(contentType, "audio/") {
			return contentType
		}
		return "application/octet-stream"
	}
}
