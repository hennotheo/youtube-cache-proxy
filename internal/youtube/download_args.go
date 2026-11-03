package youtube

import "fmt"

func buildDownloadArgs(videoID, outputTemplate string, maxBytes int64) ([]string, error) {
	if !videoIDPattern.MatchString(videoID) {
		return nil, fmt.Errorf("%w: invalid video ID", ErrInvalidVideoURL)
	}
	if validateOutputTemplate(outputTemplate, videoID) != nil || maxBytes <= 0 {
		return nil, fmt.Errorf("download output template and positive maximum size are required")
	}

	canonicalURL := "https://www.youtube.com/watch?v=" + videoID
	return []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--format", "bestaudio/best",
		"--max-filesize", fmt.Sprint(maxBytes),
		"--no-progress",
		"--part",
		"--no-overwrites",
		"--output", outputTemplate,
		canonicalURL,
	}, nil
}
