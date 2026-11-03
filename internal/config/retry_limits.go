package config

func loadRetryLimits(c *Config) error {
	if err := setEnvInt(&c.RetryLimit, "RETRY_LIMIT", 3); err != nil {
		return err
	}
	if err := setEnvInt(&c.RetryBaseSeconds, "RETRY_BASE_SECONDS", 30); err != nil {
		return err
	}
	return setEnvInt(&c.RetryMaxSeconds, "RETRY_MAX_SECONDS", 600)
}
