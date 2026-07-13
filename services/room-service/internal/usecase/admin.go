package usecase

import (
	"context"

	"github.com/worktogether/services/room-service/internal/domain"
	"github.com/worktogether/services/room-service/internal/repository"
)

func (u *RoomUsecase) AdminListRooms(ctx context.Context, isAdmin bool) ([]*domain.Room, error) {
	if !isAdmin {
		return nil, ErrUnauthorized
	}
	return u.repo.ListAdminRooms(ctx, 200)
}

func (u *RoomUsecase) AdminListAuditEvents(ctx context.Context, isAdmin bool) ([]*repository.AdminAuditEvent, error) {
	if !isAdmin {
		return nil, ErrUnauthorized
	}
	return u.repo.ListAdminAuditEvents(ctx, 100)
}

func (u *RoomUsecase) AdminListMembers(ctx context.Context, isAdmin bool, roomID string) ([]*domain.RoomMember, error) {
	if !isAdmin {
		return nil, ErrUnauthorized
	}
	return u.repo.ListMembers(ctx, roomID)
}

func (u *RoomUsecase) AdminRemoveMember(ctx context.Context, isAdmin bool, actorID, roomID, memberID string) error {
	if !isAdmin {
		return ErrUnauthorized
	}
	member, err := u.repo.GetMember(ctx, roomID, memberID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrTargetMemberNotFound
	}
	if member.RoleType == "OWNER" {
		return ErrCannotKickOwner
	}
	if err := u.repo.RemoveMember(ctx, roomID, memberID); err != nil {
		return err
	}
	u.invalidateRoomCache(ctx, roomID)
	return u.repo.CreateAdminAuditEvent(ctx, actorID, "room.member_removed", "room_member", memberID, map[string]any{"room_id": roomID})
}

func (u *RoomUsecase) AdminUpdateRoom(ctx context.Context, isAdmin bool, actorID, roomID string, input *domain.Room) (*domain.Room, error) {
	if !isAdmin {
		return nil, ErrUnauthorized
	}
	if input.Mode != "" && !isValidRoomMode(input.Mode) {
		return nil, ErrInvalidRoomMode
	}
	rm, err := u.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if rm == nil {
		return nil, ErrRoomNotFound
	}
	rm.Name, rm.Description, rm.Privacy = input.Name, input.Description, input.Privacy
	rm.AddMusicPolicy, rm.AvatarURL, rm.Rules, rm.Theme = input.AddMusicPolicy, input.AvatarURL, input.Rules, input.Theme
	if err := u.repo.UpdateRoom(ctx, rm); err != nil {
		return nil, err
	}
	if input.Mode != "" && input.Mode != rm.Mode {
		if err := u.repo.UpdateRoomMode(ctx, roomID, input.Mode); err != nil {
			return nil, err
		}
		rm.Mode = input.Mode
	}
	u.invalidateRoomCache(ctx, roomID)
	_ = u.repo.CreateAdminAuditEvent(ctx, actorID, "room.updated", "room", roomID, map[string]any{"name": rm.Name, "privacy": rm.Privacy})
	return rm, nil
}

func (u *RoomUsecase) AdminDeleteRoom(ctx context.Context, isAdmin bool, actorID, roomID string) error {
	if !isAdmin {
		return ErrUnauthorized
	}
	if err := u.repo.DeleteRoom(ctx, roomID); err != nil {
		return err
	}
	u.invalidateRoomCache(ctx, roomID)
	return u.repo.CreateAdminAuditEvent(ctx, actorID, "room.deleted", "room", roomID, map[string]any{})
}
