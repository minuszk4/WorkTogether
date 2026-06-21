package roomv1

import (
	"context"

	"google.golang.org/grpc"
)

type SetPlaybackStateRequest struct {
	RoomID string `json:"room_id"`
	Action string `json:"action"` // "pause", "resume"
}

type SetPlaybackStateResponse struct {
	Success bool `json:"success"`
}

// PlaybackInternalServiceServer is the server API for PlaybackInternalService service.
type PlaybackInternalServiceServer interface {
	SetPlaybackStateByTimer(context.Context, *SetPlaybackStateRequest) (*SetPlaybackStateResponse, error)
}

// RegisterPlaybackInternalServiceServer registers a service implementation with a gRPC server.
func RegisterPlaybackInternalServiceServer(s grpc.ServiceRegistrar, srv PlaybackInternalServiceServer) {
	s.RegisterService(&PlaybackInternalService_ServiceDesc, srv)
}

func _PlaybackInternalService_SetPlaybackStateByTimer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetPlaybackStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlaybackInternalServiceServer).SetPlaybackStateByTimer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/worktogether.playback.v1.PlaybackInternalService/SetPlaybackStateByTimer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlaybackInternalServiceServer).SetPlaybackStateByTimer(ctx, req.(*SetPlaybackStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var PlaybackInternalService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "worktogether.playback.v1.PlaybackInternalService",
	HandlerType: (*PlaybackInternalServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetPlaybackStateByTimer",
			Handler:    _PlaybackInternalService_SetPlaybackStateByTimer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "playback.proto",
}

// Client part used by timer-service
type PlaybackInternalServiceClient interface {
	SetPlaybackStateByTimer(ctx context.Context, in *SetPlaybackStateRequest, opts ...grpc.CallOption) (*SetPlaybackStateResponse, error)
}

type playbackInternalServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPlaybackInternalServiceClient(cc grpc.ClientConnInterface) PlaybackInternalServiceClient {
	return &playbackInternalServiceClient{cc}
}

func (c *playbackInternalServiceClient) SetPlaybackStateByTimer(ctx context.Context, in *SetPlaybackStateRequest, opts ...grpc.CallOption) (*SetPlaybackStateResponse, error) {
	out := new(SetPlaybackStateResponse)
	opts = append(opts, grpc.CallContentSubtype("json"))
	err := c.cc.Invoke(ctx, "/worktogether.playback.v1.PlaybackInternalService/SetPlaybackStateByTimer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
