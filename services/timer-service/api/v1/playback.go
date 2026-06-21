package v1

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

type SetPlaybackStateRequest struct {
	RoomID string `json:"room_id"`
	Action string `json:"action"` // "pause", "resume"
}

type SetPlaybackStateResponse struct {
	Success bool `json:"success"`
}

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
