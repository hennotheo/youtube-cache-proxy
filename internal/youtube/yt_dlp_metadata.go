package youtube

type ytDLPMetadata struct {
	Title          string  `json:"title"`
	Duration       float64 `json:"duration"`
	FileSize       *int64  `json:"filesize"`
	FileSizeApprox *int64  `json:"filesize_approx"`
}
