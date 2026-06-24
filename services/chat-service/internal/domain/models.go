package domain

import "time"

type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ReplyToID string    `json:"reply_to_id,omitempty"`
	IsEdited  bool      `json:"is_edited"`
	CreatedAt time.Time `json:"created_at"`
	ClientID  string    `json:"client_id,omitempty" db:"-"`
}


type MessageReaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type MessagePin struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	PinnedBy  string    `json:"pinned_by"`
	PinnedAt  time.Time `json:"pinned_at"`
}

// WebSocket structs
type WSMessage struct {
	Event  string      `json:"event"`
	RoomID string      `json:"room_id"`
	UserID string      `json:"user_id,omitempty"`
	Payload interface{} `json:"payload"`
}

type SendMessagePayload struct {
	ClientID  string   `json:"client_id,omitempty"`
	Content   string   `json:"content"`
	ReplyToID string   `json:"reply_to_id"`
	Mentions  []string `json:"mentions,omitempty"`
}

type ReactMessagePayload struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	Action    string `json:"action"` // add, remove
}

type TypingPayload struct {
	IsTyping bool `json:"is_typing"`
}
