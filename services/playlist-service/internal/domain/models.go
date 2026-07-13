package domain

import "time"

type Playlist struct {
	ID        string    `json:"id"`
	RoomID    *string   `json:"room_id,omitempty"` // Nullable if personal
	UserID    *string   `json:"user_id,omitempty"` // Nullable if room-only
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type PlaylistTrack struct {
	ID         string    `json:"id"`
	PlaylistID string    `json:"playlist_id"`
	TrackID    string    `json:"track_id"`
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Thumbnail  string    `json:"thumbnail_url"`
	DurationMS int       `json:"duration_ms"`
	SourceURL  string    `json:"source_url"`
	Position   int       `json:"position"`
	AddedBy    string    `json:"added_by"`
	Votes      int       `json:"votes"` // Calculated sum of upvotes/downvotes
	CreatedAt  time.Time `json:"created_at"`
}

type PlaylistVote struct {
	PlaylistTrackID string    `json:"playlist_track_id"`
	UserID          string    `json:"user_id"`
	VoteType        string    `json:"vote_type"` // "up" or "down"
	CreatedAt       time.Time `json:"created_at"`
}

type CreatePlaylistRequest struct {
	Name   string  `json:"name" binding:"required"`
	RoomID *string `json:"room_id"`
}

type AddTrackRequest struct {
	TrackID      string `json:"track_id" binding:"required"`
	Title        string `json:"title" binding:"required"`
	Artist       string `json:"artist"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationMS   int    `json:"duration_ms" binding:"required"`
	SourceURL    string `json:"source_url" binding:"required"`
}

type MoveTrackRequest struct {
	NewPosition *int `json:"new_position" binding:"required"`
}

type VoteTrackRequest struct {
	VoteType string `json:"vote_type" binding:"required,oneof=up down none"`
}
