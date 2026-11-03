package youtube

import "fmt"

func buildDownloadArgs(videoID, outputTemplate string) ([]string, error) {
	if !videoIDPattern.MatchString(videoID) {
		return nil, fmt.Errorf("%w: invalid video ID", ErrInvalidVideoURL)
	}
	if outputTemplate == "" {
		return nil, fmt.Errorf("download output template is required")
	}

	canonicalURL := "https://www.youtube.com/watch?v=" + videoID
	return []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--no-progress",
		"--output", outputTemplate,
		canonicalURL,
	}, nil
}
