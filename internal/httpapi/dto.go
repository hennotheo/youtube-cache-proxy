package httpapi

import "github.com/hennotheo/youtube-cache-proxy/internal/store"

type createSourceRequest struct {
	URL string `json:"url"`
}

type runRequest struct {
	Count int `json:"count"`
}

type sourceResponse struct {
	ID              int64   `json:"id"`
	VideoID         string  `json:"video_id"`
	URL             string  `json:"url"`
	Status          string  `json:"status"`
	Title           *string `json:"title,omitempty"`
	DurationSeconds *int64  `json:"duration_seconds,omitempty"`
	AudioURL        *string `json:"audio_url,omitempty"`
	AudioMime       *string `json:"audio_mime,omitempty"`
	AudioSizeBytes  *int64  `json:"audio_size_bytes,omitempty"`
	ErrorCode       *string `json:"error_code,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
	AttemptCount    int     `json:"attempt_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	StartedAt       *string `json:"started_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
}

type sourceListResponse struct {
	Items  []sourceResponse `json:"items"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type downloadRunResponse struct {
	ID             int64  `json:"id"`
	RequestedCount int    `json:"requested_count"`
	ReservedCount  int    `json:"reserved_count"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

func toSourceResponse(source store.Source) sourceResponse {
	response := sourceResponse{
		ID: source.ID, VideoID: source.VideoID, URL: source.URL, Status: source.Status,
		AttemptCount: source.AttemptCount, CreatedAt: source.CreatedAt, UpdatedAt: source.UpdatedAt,
	}
	if source.Title.Valid {
		response.Title = &source.Title.String
	}
	if source.DurationSeconds.Valid {
		response.DurationSeconds = &source.DurationSeconds.Int64
	}
	if source.Status == "ready" {
		audioURL := "/sources/" + itoa(source.ID) + "/audio"
		response.AudioURL = &audioURL
	}
	if source.AudioMime.Valid {
		response.AudioMime = &source.AudioMime.String
	}
	if source.AudioSizeBytes.Valid {
		response.AudioSizeBytes = &source.AudioSizeBytes.Int64
	}
	if source.ErrorCode.Valid {
		response.ErrorCode = &source.ErrorCode.String
	}
	if source.ErrorMessage.Valid {
		response.ErrorMessage = &source.ErrorMessage.String
	}
	if source.StartedAt.Valid {
		response.StartedAt = &source.StartedAt.String
	}
	if source.CompletedAt.Valid {
		response.CompletedAt = &source.CompletedAt.String
	}
	return response
}
