package store

// DownloadAttempt records one started download of a reserved source.
type DownloadAttempt struct {
	ID            int64
	RunID         int64
	SourceID      int64
	VideoID       string
	AttemptNumber int
	Status        string
	StartedAt     string
}
