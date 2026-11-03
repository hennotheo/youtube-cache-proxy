package store

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

func sqliteDSN(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	uriPath := filepath.ToSlash(absolute)
	if runtime.GOOS == "windows" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	// DSN flags are applied to every physical connection, including replacements.
	uri := url.URL{Scheme: "file", Path: uriPath, RawQuery: "_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL&_synchronous=FULL"}
	return uri.String(), nil
}
