package domain

import "time"

type RoomSession struct {
	ID          string     `json:"id"`
	RoomID      string     `json:"room_id"`
	CreatedBy   string     `json:"created_by"`
	Title       string     `json:"title"`
	Goal        string     `json:"goal"`
	TemplateKey string     `json:"template_key"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SessionAgendaItem struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Position  int       `json:"position"`
	IsDone    bool      `json:"is_done"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionActionItem struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	Content    string     `json:"content"`
	AssigneeID *string    `json:"assignee_id,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Status     string     `json:"status"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type SessionTimelineEvent struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	ActorID   string         `json:"actor_id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type SessionWorkspace struct {
	Session  *RoomSession            `json:"session"`
	Agenda   []*SessionAgendaItem    `json:"agenda"`
	Actions  []*SessionActionItem    `json:"actions"`
	Timeline []*SessionTimelineEvent `json:"timeline"`
}
