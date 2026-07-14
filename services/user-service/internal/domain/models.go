package domain

import "time"

type UserProfile struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url"`
	Bio          string    `json:"bio"`
	CustomStatus string    `json:"custom_status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Friendship struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	FriendID  string    `json:"friend_id"`
	Status    string    `json:"status"` // PENDING, ACCEPTED, BLOCKED
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Presence struct {
	Status     string `json:"status"` // online, offline, busy, away
	CustomText string `json:"custom_text"`
	LastActive int64  `json:"last_active"` // Unix timestamp (seconds)
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" binding:"required,max=100"`
	Bio         string `json:"bio" binding:"max=200"`
	AvatarURL   string `json:"avatar_url"`
}

type UpdateStatusRequest struct {
	Status     string `json:"status" binding:"required,oneof=online offline busy away"`
	CustomText string `json:"custom_text" binding:"max=100"`
}

type FriendRequest struct {
	FriendID string `json:"friend_id" binding:"required"`
}

type FriendResponseAction struct {
	Action string `json:"action" binding:"required,oneof=accept reject"`
}
