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
	IsMember        bool     `json:"is_member"`
	Role            string   `json:"role"`
	Permissions     []string `json:"permissions"`
	ActiveSubRoomID string   `json:"active_sub_room_id,omitempty"`
}

// RoomInternalServiceServer is the server API for RoomInternalService service.
type RoomInternalServiceServer interface {
	VerifyRoomMember(context.Context, *VerifyRoomMemberRequest) (*VerifyRoomMemberResponse, error)
}

// RegisterRoomInternalServiceServer registers a service implementation with a gRPC server.
func RegisterRoomInternalServiceServer(s grpc.ServiceRegistrar, srv RoomInternalServiceServer) {
	s.RegisterService(&RoomInternalService_ServiceDesc, srv)
}

func _RoomInternalService_VerifyRoomMember_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyRoomMemberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoomInternalServiceServer).VerifyRoomMember(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/worktogether.room.v1.RoomInternalService/VerifyRoomMember",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoomInternalServiceServer).VerifyRoomMember(ctx, req.(*VerifyRoomMemberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var RoomInternalService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "worktogether.room.v1.RoomInternalService",
	HandlerType: (*RoomInternalServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "VerifyRoomMember",
			Handler:    _RoomInternalService_VerifyRoomMember_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "room_service.proto",
}

// Client part used by chat-service / playback-service
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
