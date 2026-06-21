package grpc

import (
	"context"
	"errors"

	roomv1 "github.com/worktogether/services/playback-service/api/v1"
	"github.com/worktogether/services/playback-service/internal/usecase"
)

type PlaybackGrpcServer struct {
	uc  *usecase.PlaybackUsecase
	hub HubInterface
}

type HubInterface interface {
	TriggerTimerPlaybackAction(ctx context.Context, roomID string, action string) error
}

func NewPlaybackGrpcServer(uc *usecase.PlaybackUsecase, hub HubInterface) *PlaybackGrpcServer {
	return &PlaybackGrpcServer{uc: uc, hub: hub}
}

func (s *PlaybackGrpcServer) SetPlaybackStateByTimer(ctx context.Context, req *roomv1.SetPlaybackStateRequest) (*roomv1.SetPlaybackStateResponse, error) {
	if req.RoomID == "" || req.Action == "" {
		return &roomv1.SetPlaybackStateResponse{Success: false}, errors.New("missing parameters")
	}

	err := s.hub.TriggerTimerPlaybackAction(ctx, req.RoomID, req.Action)
	if err != nil {
		return &roomv1.SetPlaybackStateResponse{Success: false}, err
	}

	return &roomv1.SetPlaybackStateResponse{Success: true}, nil
}
