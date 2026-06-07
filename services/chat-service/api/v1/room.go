package roomv1

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "json"
}

func init() {
	if encoding.GetCodec("json") == nil {
		encoding.RegisterCodec(jsonCodec{})
	}
}

type VerifyRoomMemberRequest struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

type VerifyRoomMemberResponse struct {
	IsMember    bool     `json:"is_member"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// Client part used by chat-service to query room-service
type RoomInternalServiceClient interface {
	VerifyRoomMember(ctx context.Context, in *VerifyRoomMemberRequest, opts ...grpc.CallOption) (*VerifyRoomMemberResponse, error)
}

type roomInternalServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRoomInternalServiceClient(cc grpc.ClientConnInterface) RoomInternalServiceClient {
	return &roomInternalServiceClient{cc}
}

func (c *roomInternalServiceClient) VerifyRoomMember(ctx context.Context, in *VerifyRoomMemberRequest, opts ...grpc.CallOption) (*VerifyRoomMemberResponse, error) {
	out := new(VerifyRoomMemberResponse)
	opts = append(opts, grpc.CallContentSubtype("json"))
	err := c.cc.Invoke(ctx, "/worktogether.room.v1.RoomInternalService/VerifyRoomMember", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

