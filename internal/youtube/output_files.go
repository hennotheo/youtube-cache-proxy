package youtube

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const outputExtensionTemplate = "%(ext)s"

func validateOutputTemplate(outputTemplate, videoID string) error {
	if !videoIDPattern.MatchString(videoID) || filepath.Base(outputTemplate) != videoID+"."+outputExtensionTemplate {
		return fmt.Errorf("output template must use the video ID and one %s token", outputExtensionTemplate)
	}
	return nil
}

func outputFiles(outputTemplate string, includePartials bool) ([]string, error) {
	videoID, err := outputVideoID(outputTemplate)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(outputTemplate)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	prefix := videoID + "."
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, prefix) || name == prefix {
			continue
		}
		isPartial := strings.HasSuffix(name, ".part")
		if isPartial != includePartials {
			continue
		}
		files = append(files, filepath.Join(directory, name))
	}
	return files, nil
}

func outputVideoID(outputTemplate string) (string, error) {
	base := filepath.Base(outputTemplate)
	if !strings.HasSuffix(base, "."+outputExtensionTemplate) {
		return "", fmt.Errorf("output template must end with video ID and extension token")
	}
	videoID := strings.TrimSuffix(base, "."+outputExtensionTemplate)
	if !videoIDPattern.MatchString(videoID) {
		return "", fmt.Errorf("output template must contain a valid video ID")
	}
	return videoID, nil
}

func cleanupPartialFiles(outputTemplate string) error {
	files, err := outputFiles(outputTemplate, true)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var cleanupErr error
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func partialOutputSize(outputTemplate string) (int64, error) {
	files, err := outputFiles(outputTemplate, true)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var size int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		size += info.Size()
	}
	return size, nil
}

func actualOutputSize(outputTemplate string) (int64, error) {
	files, err := outputFiles(outputTemplate, false)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, fmt.Errorf("yt-dlp completed without an output file")
	}
	var size int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return 0, err
		}
		size += info.Size()
	}
	return size, nil
}

func publishOutput(stagingTemplate, finalTemplate string) error {
	stagedFiles, err := outputFiles(stagingTemplate, false)
	if err != nil {
		return err
	}
	if len(stagedFiles) != 1 {
		return fmt.Errorf("expected one completed audio file, found %d", len(stagedFiles))
	}
	finalDirectory := filepath.Dir(finalTemplate)
	if err := os.MkdirAll(finalDirectory, 0700); err != nil {
		return fmt.Errorf("prepare audio directory: %w", err)
	}
	stagingID, err := outputVideoID(stagingTemplate)
	if err != nil {
		return err
	}
	finalID, err := outputVideoID(finalTemplate)
	if err != nil {
		return err
	}
	stagedName := filepath.Base(stagedFiles[0])
	extension := strings.TrimPrefix(stagedName, stagingID+".")
	finalPath := filepath.Join(finalDirectory, finalID+"."+extension)
	if _, err := os.Lstat(finalPath); err == nil {
		return fmt.Errorf("audio output already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stagedFiles[0], finalPath); err != nil {
		return fmt.Errorf("publish audio output: %w", err)
	}
	return nil
}

func removeOutputFiles(outputTemplate string) error {
	partials, err := outputFiles(outputTemplate, true)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	files, err := outputFiles(outputTemplate, false)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var cleanupErr error
	for _, file := range append(partials, files...) {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}
