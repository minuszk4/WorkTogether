package v1

import (
	"context"

	"google.golang.org/grpc"
)

type VerifyRoomMemberRequest struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

type VerifyRoomMemberResponse struct {
	IsMember    bool     `json:"is_member"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type RoomInternalServiceClient interface {
	VerifyRoomMember(context.Context, *VerifyRoomMemberRequest, ...grpc.CallOption) (*VerifyRoomMemberResponse, error)
}

type roomInternalServiceClient struct{ cc grpc.ClientConnInterface }

func NewRoomInternalServiceClient(cc grpc.ClientConnInterface) RoomInternalServiceClient {
	return &roomInternalServiceClient{cc: cc}
}

func (c *roomInternalServiceClient) VerifyRoomMember(ctx context.Context, in *VerifyRoomMemberRequest, opts ...grpc.CallOption) (*VerifyRoomMemberResponse, error) {
	out := new(VerifyRoomMemberResponse)
	opts = append(opts, grpc.CallContentSubtype("json"))
	if err := c.cc.Invoke(ctx, "/worktogether.room.v1.RoomInternalService/VerifyRoomMember", in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}
