package config

func loadDownloadLimits(c *Config) error {
	for _, setting := range []func() error{
		func() error { return setEnvInt(&c.Concurrency, "DOWNLOAD_CONCURRENCY", 1) },
		func() error { return setEnvInt(&c.PauseMinSeconds, "PAUSE_MIN_SECONDS", 10) },
		func() error { return setEnvInt(&c.PauseMaxSeconds, "PAUSE_MAX_SECONDS", 30) },
		func() error { return setEnvInt(&c.MaxAttemptsPer24Hours, "MAX_ATTEMPTS_PER_24H", 50) },
		func() error { return setEnvInt(&c.MaxRateBytesPerSecond, "MAX_RATE_BYTES_PER_SECOND", 5_000_000) },
		func() error { return setEnvInt(&c.MaxAudioBytes, "MAX_AUDIO_BYTES", 500_000_000) },
		func() error { return setEnvInt(&c.MaxVideoDurationSeconds, "MAX_VIDEO_DURATION_SECONDS", 14_400) },
		func() error { return setEnvInt(&c.DownloadTimeoutSeconds, "DOWNLOAD_TIMEOUT_SECONDS", 14_400) },
	} {
		if err := setting(); err != nil {
			return err
		}
	}
	return nil
}
