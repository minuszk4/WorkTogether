package domain

import "time"

type Notification struct {
	ID         string    `json:"id"`
	ReceiverID string    `json:"receiver_id"`
	SenderID   string    `json:"sender_id"`
	Type       string    `json:"type"` // "mention", "friend_request", "room_invite", "room_event"
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

type TriggerNotificationRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Content    string `json:"content" binding:"required"`
}
