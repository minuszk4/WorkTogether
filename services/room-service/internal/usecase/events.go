package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/worktogether/services/room-service/internal/domain"
)

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
