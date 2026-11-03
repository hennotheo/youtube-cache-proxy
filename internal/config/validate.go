package config

import (
	"crypto/subtle"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func (c Config) validate() error {
	if err := validateAddress(c.HTTPAddr); err != nil {
		return err
	}
	if err := validateSecret("PRODUCER_TOKEN", c.ProducerToken); err != nil {
		return err
	}
	if err := validateSecret("ADMIN_TOKEN", c.AdminToken); err != nil {
		return err
	}
	if err := validateSecret("SESSION_SECRET", c.SessionSecret); err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(c.ProducerToken), []byte(c.AdminToken)) == 1 {
		return fmt.Errorf("ADMIN_TOKEN must differ from PRODUCER_TOKEN")
	}
	if subtle.ConstantTimeCompare([]byte(c.SessionSecret), []byte(c.ProducerToken)) == 1 ||
		subtle.ConstantTimeCompare([]byte(c.SessionSecret), []byte(c.AdminToken)) == 1 {
		return fmt.Errorf("SESSION_SECRET must differ from API tokens")
	}
	if err := c.validateLimits(); err != nil {
		return err
	}
	return c.validatePaths()
}

func validateSecret(name, value string) error {
	if len(value) < 32 || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must contain at least 32 non-whitespace bytes", name)
	}
	return nil
}

func validateAddress(address string) error {
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR must be a host:port address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("HTTP_ADDR must contain a valid port")
	}
	return nil
}

func (c Config) validateLimits() error {
	if c.Concurrency != 1 {
		return fmt.Errorf("DOWNLOAD_CONCURRENCY must be 1")
	}
	if c.PauseMinSeconds < 0 || c.PauseMaxSeconds < c.PauseMinSeconds {
		return fmt.Errorf("PAUSE_MIN_SECONDS and PAUSE_MAX_SECONDS must form a valid range")
	}
	for _, err := range []error{
		validatePositive("MAX_ATTEMPTS_PER_24H", c.MaxAttemptsPer24Hours),
		validatePositive("MAX_RATE_BYTES_PER_SECOND", c.MaxRateBytesPerSecond),
		validatePositive("MAX_AUDIO_BYTES", c.MaxAudioBytes),
		validatePositive("MAX_VIDEO_DURATION_SECONDS", c.MaxVideoDurationSeconds),
		validatePositive("RETRY_LIMIT", c.RetryLimit),
		validatePositive("RETRY_BASE_SECONDS", c.RetryBaseSeconds),
		validatePositive("RETRY_MAX_SECONDS", c.RetryMaxSeconds),
	} {
		if err != nil {
			return err
		}
	}
	if c.RetryMaxSeconds < c.RetryBaseSeconds {
		return fmt.Errorf("RETRY_MAX_SECONDS must be at least RETRY_BASE_SECONDS")
	}
	return nil
}

func validatePositive(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}
