package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setValidEnvironment(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	t.Setenv("SQLITE_PATH", filepath.Join(root, "db", "queue.db"))
	t.Setenv("AUDIO_DIR", filepath.Join(root, "audio"))
	t.Setenv("PRODUCER_TOKEN", strings.Repeat("p", 32))
	t.Setenv("ADMIN_TOKEN", strings.Repeat("a", 32))
	t.Setenv("SESSION_SECRET", strings.Repeat("s", 32))
}

func TestLoadUsesDownloadDefaults(t *testing.T) {
	setValidEnvironment(t)
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.Concurrency != 1 || config.PauseMinSeconds != 10 || config.PauseMaxSeconds != 30 || config.MaxAttemptsPer24Hours != 50 {
		t.Fatal("unexpected worker defaults")
	}
	if config.MaxRateBytesPerSecond != 5_000_000 || config.MaxAudioBytes != 500_000_000 || config.MaxVideoDurationSeconds != 14_400 || config.DownloadTimeoutSeconds != 14_400 {
		t.Fatal("unexpected media defaults")
	}
	if config.RetryLimit != 3 || config.RetryBaseSeconds != 30 || config.RetryMaxSeconds != 600 {
		t.Fatal("unexpected retry defaults")
	}
	if _, err := os.Stat(filepath.Dir(config.SQLitePath)); err != nil {
		t.Fatalf("SQLite directory not prepared: %v", err)
	}
	if _, err := os.Stat(config.AudioDir); err != nil {
		t.Fatalf("audio directory not prepared: %v", err)
	}
}

func TestLoadRejectsInvalidDownloadTimeout(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("DOWNLOAD_TIMEOUT_SECONDS", "0")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DOWNLOAD_TIMEOUT_SECONDS") {
		t.Fatalf("expected download timeout validation error, got %v", err)
	}
}

func TestLoadRejectsMissingAndDuplicateSecretsWithoutLeakingThem(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("ADMIN_TOKEN", strings.Repeat("p", 32))
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_TOKEN") {
		t.Fatalf("expected ADMIN_TOKEN validation error, got %v", err)
	}
	if strings.Contains(err.Error(), strings.Repeat("p", 32)) {
		t.Fatal("secret leaked in validation error")
	}

	t.Setenv("ADMIN_TOKEN", "")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_TOKEN") {
		t.Fatalf("expected missing ADMIN_TOKEN error, got %v", err)
	}
}

func TestLoadRejectsInvalidLimitsAndPaths(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("PAUSE_MIN_SECONDS", "31")
	t.Setenv("PAUSE_MAX_SECONDS", "30")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid pause range to fail")
	}

	t.Setenv("PAUSE_MIN_SECONDS", "10")
	root := t.TempDir()
	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUDIO_DIR", file)
	if _, err := Load(); err == nil {
		t.Fatal("expected inaccessible audio path to fail")
	}
}

func TestLoadReadsOverridesAndRejectsMalformedValues(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("MAX_ATTEMPTS_PER_24H", "12")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxAttemptsPer24Hours != 12 {
		t.Fatalf("attempt limit = %d, want 12", config.MaxAttemptsPer24Hours)
	}

	t.Setenv("MAX_ATTEMPTS_PER_24H", "twelve")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MAX_ATTEMPTS_PER_24H") {
		t.Fatalf("expected named integer validation error, got %v", err)
	}
}

func TestLoadRejectsSessionSecretMatchingAPIToken(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("SESSION_SECRET", strings.Repeat("a", 32))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Fatalf("expected distinct session secret error, got %v", err)
	}
}

func TestLoadRejectsSQLitePathThatIsADirectory(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("SQLITE_PATH", t.TempDir())
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SQLITE_PATH") {
		t.Fatalf("expected SQLite file path error, got %v", err)
	}
}
