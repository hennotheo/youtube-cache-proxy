package youtube

import "fmt"

func buildMetadataArgs(videoID string) ([]string, error) {
	if !videoIDPattern.MatchString(videoID) {
		return nil, fmt.Errorf("%w: invalid video ID", ErrInvalidVideoURL)
	}

	return []string{
		"--ignore-config",
		"--no-cookies",
		"--no-cookies-from-browser",
		"--no-playlist",
		"--format", "bestaudio/best",
		"--dump-single-json",
		"--skip-download",
		"https://www.youtube.com/watch?v=" + videoID,
	}, nil
}
