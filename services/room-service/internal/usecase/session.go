package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/worktogether/services/room-service/internal/domain"
)

var (
	ErrActiveSessionExists = errors.New("phòng đã có session đang diễn ra")
	ErrSessionNotFound     = errors.New("không tìm thấy session")
)

type StartSessionInput struct {
	Title       string
	Goal        string
	TemplateKey string
}

type AddAgendaInput struct {
	Content  string
	Position int
}

type AddActionInput struct {
	Content    string
	AssigneeID *string
	DueAt      *time.Time
}

func (u *RoomUsecase) StartSession(ctx context.Context, userID, roomID string, input *StartSessionInput) (*domain.RoomSession, error) {
	if err := u.requireHost(ctx, userID, roomID); err != nil {
		return nil, err
	}
	active, err := u.repo.GetActiveSession(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrActiveSessionExists
	}
	session := &domain.RoomSession{RoomID: roomID, CreatedBy: userID, Title: input.Title, Goal: input.Goal, TemplateKey: input.TemplateKey}
	if err := u.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	if err := u.recordSessionEvent(ctx, session.ID, userID, "session.started", map[string]any{"title": session.Title, "template_key": session.TemplateKey}); err != nil {
		return nil, err
	}
	if mode := roomModeForTemplate(session.TemplateKey); mode != "" {
		if err := u.repo.UpdateRoomMode(ctx, roomID, mode); err != nil {
			return nil, err
		}
		u.invalidateRoomCache(ctx, roomID)
		u.publishRoomEvent(ctx, roomID, "room:mode_changed", map[string]string{"mode": mode, "changed_by": userID})
	}
	u.publishRoomEvent(ctx, roomID, "session:started", map[string]any{"session": session})
	return session, nil
}

func roomModeForTemplate(templateKey string) string {
	switch templateKey {
	case "focus", "study":
		return "focus"
	case "standup":
		return "collaborate"
	case "chill":
		return "chill"
	default:
		return ""
	}
}

func (u *RoomUsecase) GetActiveSession(ctx context.Context, userID, roomID string) (*domain.RoomSession, error) {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}
	return u.repo.GetActiveSession(ctx, roomID)
}

func (u *RoomUsecase) CompleteSession(ctx context.Context, userID, roomID, sessionID string) error {
	if err := u.requireHost(ctx, userID, roomID); err != nil {
		return err
	}
	if session, err := u.repo.GetSession(ctx, roomID, sessionID); err != nil {
		return err
	} else if session == nil {
		return ErrSessionNotFound
	}
	if err := u.repo.CompleteSession(ctx, roomID, sessionID); err != nil {
		return err
	}
	if err := u.recordSessionEvent(ctx, sessionID, userID, "session.completed", map[string]any{}); err != nil {
		return err
	}
	u.publishRoomEvent(ctx, roomID, "session:completed", map[string]string{"session_id": sessionID})
	return nil
}

func (u *RoomUsecase) GetSessionWorkspace(ctx context.Context, userID, roomID, sessionID string) (*domain.SessionWorkspace, error) {
	if _, err := u.requireMember(ctx, userID, roomID); err != nil {
		return nil, err
	}
	session, err := u.repo.GetSession(ctx, roomID, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	agenda, err := u.repo.ListAgendaItems(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	actions, err := u.repo.ListActionItems(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	timeline, err := u.repo.ListSessionTimeline(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &domain.SessionWorkspace{Session: session, Agenda: agenda, Actions: actions, Timeline: timeline}, nil
}

func (u *RoomUsecase) AddAgendaItem(ctx context.Context, userID, roomID, sessionID string, input *AddAgendaInput) (*domain.SessionAgendaItem, error) {
	if _, err := u.requireMember(ctx, userID, roomID); err != nil {
		return nil, err
	}
	if session, err := u.repo.GetSession(ctx, roomID, sessionID); err != nil {
		return nil, err
	} else if session == nil {
		return nil, ErrSessionNotFound
	}
	item := &domain.SessionAgendaItem{SessionID: sessionID, Content: input.Content, Position: input.Position}
	if err := u.repo.CreateAgendaItem(ctx, item); err != nil {
		return nil, err
	}
	if err := u.recordSessionEvent(ctx, sessionID, userID, "agenda.added", map[string]any{"item_id": item.ID}); err != nil {
		return nil, err
	}
	u.publishRoomEvent(ctx, roomID, "session:agenda_added", item)
	return item, nil
}

func (u *RoomUsecase) AddActionItem(ctx context.Context, userID, roomID, sessionID string, input *AddActionInput) (*domain.SessionActionItem, error) {
	if _, err := u.requireMember(ctx, userID, roomID); err != nil {
		return nil, err
	}
	if session, err := u.repo.GetSession(ctx, roomID, sessionID); err != nil {
		return nil, err
	} else if session == nil {
		return nil, ErrSessionNotFound
	}
	item := &domain.SessionActionItem{SessionID: sessionID, Content: input.Content, AssigneeID: input.AssigneeID, DueAt: input.DueAt, CreatedBy: userID}
	if err := u.repo.CreateActionItem(ctx, item); err != nil {
		return nil, err
	}
	if err := u.recordSessionEvent(ctx, sessionID, userID, "action.added", map[string]any{"item_id": item.ID}); err != nil {
		return nil, err
	}
	u.publishRoomEvent(ctx, roomID, "session:action_added", item)
	return item, nil
}

func (u *RoomUsecase) SetAgendaItemDone(ctx context.Context, userID, roomID, sessionID, itemID string, isDone bool) error {
	if _, err := u.requireMember(ctx, userID, roomID); err != nil {
		return err
	}
	if session, err := u.repo.GetSession(ctx, roomID, sessionID); err != nil {
		return err
	} else if session == nil {
		return ErrSessionNotFound
	}
	if err := u.repo.UpdateAgendaItem(ctx, sessionID, itemID, isDone); err != nil {
		return err
	}
	u.publishRoomEvent(ctx, roomID, "session:agenda_updated", map[string]any{"id": itemID, "is_done": isDone})
	return nil
}

func (u *RoomUsecase) SetActionStatus(ctx context.Context, userID, roomID, sessionID, itemID, status string) error {
	if status != "OPEN" && status != "DONE" {
		return ErrUnauthorized
	}
	if _, err := u.requireMember(ctx, userID, roomID); err != nil {
		return err
	}
	if session, err := u.repo.GetSession(ctx, roomID, sessionID); err != nil {
		return err
	} else if session == nil {
		return ErrSessionNotFound
	}
	if err := u.repo.UpdateActionStatus(ctx, sessionID, itemID, status); err != nil {
		return err
	}
	u.publishRoomEvent(ctx, roomID, "session:action_updated", map[string]string{"id": itemID, "status": status})
	return nil
}

func (u *RoomUsecase) GetPersonalActionItems(ctx context.Context, userID string) ([]*domain.PersonalActionItem, error) {
	return u.repo.ListPersonalActionItems(ctx, userID)
}

func (u *RoomUsecase) requireHost(ctx context.Context, userID, roomID string) error {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if member == nil || (member.RoleType != "OWNER" && member.RoleType != "MODERATOR") {
		return ErrUnauthorized
	}
	return nil
}

func (u *RoomUsecase) requireMember(ctx context.Context, userID, roomID string) (*domain.RoomMember, error) {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}
	return member, nil
}

func (u *RoomUsecase) recordSessionEvent(ctx context.Context, sessionID, actorID, eventType string, payload map[string]any) error {
	return u.repo.CreateSessionTimelineEvent(ctx, &domain.SessionTimelineEvent{SessionID: sessionID, ActorID: actorID, EventType: eventType, Payload: payload})
}

func (u *RoomUsecase) publishRoomEvent(ctx context.Context, roomID, event string, payload any) {
	if u.rdb == nil {
		return
	}
	message, err := json.Marshal(map[string]any{"event": event, "room_id": roomID, "payload": payload})
	if err == nil {
		u.rdb.Publish(ctx, "ch:chat:"+roomID, message)
	}
}
