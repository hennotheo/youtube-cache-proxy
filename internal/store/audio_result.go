package store

// AudioResult contains the metadata written when a download succeeds.
type AudioResult struct {
	Title           string
	DurationSeconds int64
	Path            string
	Mime            string
	SizeBytes       int64
}
