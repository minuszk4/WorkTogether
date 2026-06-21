package domain

import "time"

type Room struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Privacy        string    `json:"privacy"` // PUBLIC, PRIVATE, FRIENDS
	PasswordHash   string    `json:"-"`
	InviteCode     string    `json:"invite_code"`
	OwnerID        string    `json:"owner_id"`
	AddMusicPolicy string    `json:"add_music_policy"`
	ParentID       *string   `json:"parent_id,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	Rules          *string   `json:"rules,omitempty"`
	Theme          string    `json:"theme"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RoomRole struct {
	ID                 string    `json:"id"`
	RoomID             string    `json:"room_id"`
	Name               string    `json:"name"`
	CanChat            bool      `json:"can_chat"`
	CanManagePlaylist  bool      `json:"can_manage_playlist"`
	CanControlPlayback bool      `json:"can_control_playback"`
	CanModerateMembers bool      `json:"can_moderate_members"`
	CanUseVoice        bool      `json:"can_use_voice"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type RoomMember struct {
	ID              string     `json:"id"`
	RoomID          string     `json:"room_id"`
	UserID          string     `json:"user_id"`
	RoleID          string     `json:"role_id,omitempty"`   // custom role
	RoleType        string     `json:"role_type"`           // OWNER, MODERATOR, MEMBER
	ActiveSubRoomID *string    `json:"active_sub_room_id,omitempty"`
	MutedUntil      *time.Time `json:"muted_until,omitempty"`
	JoinedAt        time.Time  `json:"joined_at"`
}

type RoomBan struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	BannedBy  string    `json:"banned_by"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=100"`
	Description string `json:"description" binding:"max=500"`
	Privacy     string `json:"privacy" binding:"required,oneof=public private friends"`
	Password    string `json:"password"`
}

type JoinRoomRequest struct {
	Password string `json:"password"`
}

type CreateRoleRequest struct {
	Name               string `json:"name" binding:"required,max=50"`
	CanChat            bool   `json:"can_chat"`
	CanManagePlaylist  bool   `json:"can_manage_playlist"`
	CanControlPlayback bool   `json:"can_control_playback"`
	CanModerateMembers bool   `json:"can_moderate_members"`
	CanUseVoice        bool   `json:"can_use_voice"`
}

type AssignRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

type BanRequest struct {
	Reason string `json:"reason"`
}

type MuteRequest struct {
	DurationSeconds int `json:"duration_seconds" binding:"required,min=10"`
}
