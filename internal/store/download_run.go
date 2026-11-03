package store

// DownloadRun records one manually requested batch of sources.
type DownloadRun struct {
	ID             int64
	RequestedCount int
	ReservedCount  int
	Status         string
	CreatedAt      string
}
