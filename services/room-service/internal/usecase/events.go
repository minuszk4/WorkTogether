package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/room-service/internal/domain"
)

func (u *RoomUsecase) StartEventReminderWorker(ctx context.Context) {
	if u.rdb == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			u.publishDueEventReminders(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (u *RoomUsecase) publishDueEventReminders(ctx context.Context) {
	events, err := u.repo.ClaimDueRoomEvents(ctx)
	if err != nil {
		return
	}
	for _, event := range events {
		members, err := u.repo.ListMembers(ctx, event.RoomID)
		if err != nil {
			continue
		}
		for _, member := range members {
			payload, _ := json.Marshal(map[string]string{"receiver_id": member.UserID, "sender_id": event.CreatedBy, "type": "room_event_reminder", "content": event.Title})
			_ = u.rdb.XAdd(ctx, &redis.XAddArgs{Stream: "stream:notification_trigger", Values: map[string]any{"payload": string(payload)}}).Err()
		}
	}
}

func (u *RoomUsecase) CreateRoomEvent(ctx context.Context, userID, roomID, title, description string, startsAt time.Time) (*domain.RoomEvent, error) {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}
	if member.RoleType != "OWNER" && !hasRoomPermission(u.ResolvePermissions(ctx, member), "CAN_MODERATE_MEMBERS") {
		return nil, ErrUnauthorized
	}
	if startsAt.Before(time.Now()) {
		return nil, ErrUnauthorized
	}
	event := &domain.RoomEvent{RoomID: roomID, CreatedBy: userID, Title: title, Description: description, StartsAt: startsAt}
	if err := u.repo.CreateRoomEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func hasRoomPermission(permissions []string, expected string) bool {
	for _, permission := range permissions {
		if strings.EqualFold(permission, expected) {
			return true
		}
	}
	return false
}

func (u *RoomUsecase) ListUpcomingRoomEvents(ctx context.Context, userID, roomID string) ([]*domain.RoomEvent, error) {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}
	return u.repo.ListUpcomingRoomEvents(ctx, roomID)
}

func (u *RoomUsecase) CancelRoomEvent(ctx context.Context, userID, roomID, eventID string) error {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrNotMember
	}
	if member.RoleType != "OWNER" && !hasRoomPermission(u.ResolvePermissions(ctx, member), "CAN_MODERATE_MEMBERS") {
		return ErrUnauthorized
	}
	return u.repo.CancelRoomEvent(ctx, roomID, eventID)
}
