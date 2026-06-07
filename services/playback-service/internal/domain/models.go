package domain

type PlaybackState struct {
	State          string `json:"state"` // "playing", "paused", "stopped"
	CurrentTrackID string `json:"current_track_id"`
	PositionMS     int    `json:"position_ms"`
	UpdatedAt      int64  `json:"updated_at"` // Server timestamp in milliseconds
	Title          string `json:"title"`
	Artist         string `json:"artist"`
	ThumbnailURL   string `json:"thumbnail_url"`
	DurationMS     int    `json:"duration_ms"`
	SourceURL      string `json:"source_url"`
}

type WSMessage struct {
	Event   string      `json:"event"`
	RoomID  string      `json:"room_id"`
	Payload interface{} `json:"payload"`
}

type PingPayload struct {
	T1 int64 `json:"t1"`
}

type PongPayload struct {
	T1 int64 `json:"t1"`
	T2 int64 `json:"t2"`
	T3 int64 `json:"t3"`
}

type ControlPayload struct {
	Action       string `json:"action"` // "play", "pause", "seek"
	TrackID      string `json:"track_id"`
	PositionMS   int    `json:"position_ms"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationMS   int    `json:"duration_ms"`
	SourceURL    string `json:"source_url"`
}
