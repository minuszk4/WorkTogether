package domain

import "time"

type Track struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Artist       string    `json:"artist"`
	ThumbnailURL string    `json:"thumbnail_url"`
	DurationMS   int       `json:"duration_ms"`
	Source       string    `json:"source"` // "youtube" or "upload"
	SourceURL    string    `json:"source_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type PlaybackHistory struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	TrackID   string    `json:"track_id"`
	PlayedAt  time.Time `json:"played_at"`
}

type AddTrackRequest struct {
	SourceURL string `json:"source_url" binding:"required"`
}

type TrackLyrics struct {
	TrackID   string    `json:"track_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Bookmark struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"room_id"`
	UserID     string    `json:"user_id"`
	TrackID    string    `json:"track_id"`
	PositionMS int       `json:"position_ms"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}
