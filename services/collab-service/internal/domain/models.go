package domain

import (
	"encoding/json"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoteBlock struct {
	ID         string    `json:"id"`
	NoteID     string    `json:"note_id"`
	BlockType  string    `json:"block_type"` // "text" or "todo"
	Content    string    `json:"content"`
	IsChecked  bool      `json:"is_checked"`
	OrderIndex int       `json:"order_index"`
	UpdatedBy  string    `json:"updated_by"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WhiteboardSnapshot struct {
	RoomID    string          `json:"room_id"`
	Snapshot  json.RawMessage `json:"snapshot"`
	UpdatedAt time.Time       `json:"updated_at"`
}
