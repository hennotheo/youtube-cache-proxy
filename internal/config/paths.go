package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func (c Config) validatePaths() error {
	if filepath.Base(c.SQLitePath) == "." || filepath.Base(c.SQLitePath) == string(filepath.Separator) {
		return fmt.Errorf("SQLITE_PATH must name a file")
	}
	if info, err := os.Stat(c.SQLitePath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("SQLITE_PATH must name a file")
		}
		file, err := os.OpenFile(c.SQLitePath, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("SQLITE_PATH file is not writable: %w", err)
		}
		_ = file.Close()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("SQLITE_PATH file is not accessible: %w", err)
	}
	if err := prepareDirectory("SQLITE_PATH", filepath.Dir(c.SQLitePath)); err != nil {
		return err
	}
	return prepareDirectory("AUDIO_DIR", c.AudioDir)
}

func prepareDirectory(name, path string) error {
	if path == "" {
		return fmt.Errorf("%s must name a directory", name)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("%s directory is not accessible: %w", name, err)
	}
	file, err := os.CreateTemp(path, ".write-check-*")
	if err != nil {
		return fmt.Errorf("%s directory is not writable: %w", name, err)
	}
	_ = file.Close()
	_ = os.Remove(file.Name())
	return nil
}
