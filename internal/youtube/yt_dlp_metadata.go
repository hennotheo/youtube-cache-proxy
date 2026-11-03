package youtube

type ytDLPMetadata struct {
	Title          string   `json:"title"`
	Duration       float64  `json:"duration"`
	FileSize       *int64   `json:"filesize"`
	FileSizeApprox *int64   `json:"filesize_approx"`
	Uploader       string   `json:"uploader"`
	UploaderID     string   `json:"uploader_id"`
	UploaderURL    string   `json:"uploader_url"`
	Channel        string   `json:"channel"`
	ChannelID      string   `json:"channel_id"`
	ChannelURL     string   `json:"channel_url"`
	Thumbnail      string   `json:"thumbnail"`
	UploadDate     string   `json:"upload_date"`
	ReleaseDate    string   `json:"release_date"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Categories     []string `json:"categories"`
	ViewCount      *int64   `json:"view_count"`
	Language       string   `json:"language"`
	LiveStatus     string   `json:"live_status"`
}
