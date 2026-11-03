package config

import (
	"fmt"
	"os"
	"strconv"
)

// Load reads and validates the service configuration from the environment.
func Load() (Config, error) {
	c := Config{
		HTTPAddr:      envString("HTTP_ADDR", "127.0.0.1:8080"),
		SQLitePath:    envString("SQLITE_PATH", "data/queue.db"),
		AudioDir:      envString("AUDIO_DIR", "data/audio"),
		ProducerToken: os.Getenv("PRODUCER_TOKEN"),
		AdminToken:    os.Getenv("ADMIN_TOKEN"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}
	if err := loadDownloadLimits(&c); err != nil {
		return Config{}, err
	}
	if err := loadRetryLimits(&c); err != nil {
		return Config{}, err
	}
	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func envString(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}

func setEnvInt(target *int, name string, fallback int) error {
	value, err := envInt(name, fallback)
	if err != nil {
		return err
	}
	*target = value
	return nil
}
