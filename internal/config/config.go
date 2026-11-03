package config

// Config contains the runtime settings supplied through environment variables.
type Config struct {
	HTTPAddr                string
	SQLitePath              string
	AudioDir                string
	ProducerToken           string
	AdminToken              string
	SessionSecret           string
	Concurrency             int
	PauseMinSeconds         int
	PauseMaxSeconds         int
	MaxAttemptsPer24Hours   int
	MaxRateBytesPerSecond   int
	MaxAudioBytes           int
	MaxVideoDurationSeconds int
	RetryLimit              int
	RetryBaseSeconds        int
	RetryMaxSeconds         int
}
