package grpc

import (
	"context"
	"time"

	roomv1 "github.com/worktogether/services/room-service/api/v1"
	"github.com/worktogether/services/room-service/internal/usecase"
)

type RoomGrpcServer struct {
	usecase *usecase.RoomUsecase
}

func NewRoomGrpcServer(uc *usecase.RoomUsecase) *RoomGrpcServer {
	return &RoomGrpcServer{usecase: uc}
}

func (s *RoomGrpcServer) VerifyRoomMember(ctx context.Context, req *roomv1.VerifyRoomMemberRequest) (*roomv1.VerifyRoomMemberResponse, error) {
	m, err := s.usecase.GetMember(ctx, req.RoomID, req.UserID)
	if err != nil || m == nil {
		return &roomv1.VerifyRoomMemberResponse{
			IsMember: false,
		}, nil
	}

	// Xác định danh sách permissions
	permissions := []string{}
	if m.RoleType == "OWNER" {
		permissions = append(permissions, "CAN_CHAT", "CAN_MANAGE_PLAYLIST", "CAN_CONTROL_PLAYBACK", "CAN_MODERATE_MEMBERS", "CAN_USE_VOICE")
	} else if m.RoleType == "MODERATOR" {
		permissions = append(permissions, "CAN_CHAT", "CAN_MANAGE_PLAYLIST", "CAN_CONTROL_PLAYBACK", "CAN_MODERATE_MEMBERS", "CAN_USE_VOICE")
	} else {
		// Member mặc định
		permissions = append(permissions, "CAN_CHAT", "CAN_USE_VOICE")
	}

	// Nếu có custom role, bổ sung/override permissions
	if m.RoleID != "" {
		role, err := s.usecase.GetRoleByID(ctx, m.RoleID)
		if err == nil && role != nil {
			permissions = []string{} // Reset để map theo custom
			if role.CanChat {
				permissions = append(permissions, "CAN_CHAT")
			}
			if role.CanManagePlaylist {
				permissions = append(permissions, "CAN_MANAGE_PLAYLIST")
			}
			if role.CanControlPlayback {
				permissions = append(permissions, "CAN_CONTROL_PLAYBACK")
			}
			if role.CanModerateMembers {
				permissions = append(permissions, "CAN_MODERATE_MEMBERS")
			}
			if role.CanUseVoice {
				permissions = append(permissions, "CAN_USE_VOICE")
			}
		}
	}

	if m.MutedUntil != nil && m.MutedUntil.After(time.Now()) {
		filtered := []string{}
		for _, p := range permissions {
			if p != "CAN_CHAT" && p != "CAN_USE_VOICE" {
				filtered = append(filtered, p)
			}
		}
		permissions = filtered
	}

	activeSubRoomID := ""
	if m.ActiveSubRoomID != nil {
		activeSubRoomID = *m.ActiveSubRoomID
	}

	return &roomv1.VerifyRoomMemberResponse{
		IsMember:        true,
		Role:            m.RoleType,
		Permissions:     permissions,
		ActiveSubRoomID: activeSubRoomID,
	}, nil
}
