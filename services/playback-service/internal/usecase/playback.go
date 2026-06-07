package usecase

import (
	"context"
	"time"

	"github.com/worktogether/services/playback-service/internal/domain"
	"github.com/worktogether/services/playback-service/internal/repository"
)

type PlaybackUsecase struct {
	redisRepo *repository.RedisRepository
}

func NewPlaybackUsecase(rdb *repository.RedisRepository) *PlaybackUsecase {
	return &PlaybackUsecase{redisRepo: rdb}
}

func (u *PlaybackUsecase) GetOrCreateState(ctx context.Context, roomID string) (*domain.PlaybackState, error) {
	state, err := u.redisRepo.GetPlaybackState(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if state == nil {
		// Trả về trạng thái mặc định ban đầu
		state = &domain.PlaybackState{
			State:          "stopped",
			CurrentTrackID: "",
			PositionMS:     0,
			UpdatedAt:      time.Now().UnixNano() / int64(time.Millisecond),
			Title:          "",
			Artist:         "",
			ThumbnailURL:   "",
			DurationMS:     0,
			SourceURL:      "",
		}
	}

	return state, nil
}

func (u *PlaybackUsecase) UpdateState(ctx context.Context, roomID string, req *domain.ControlPayload) (*domain.PlaybackState, error) {
	nowMS := time.Now().UnixNano() / int64(time.Millisecond)

	stateStr := "playing"
	if req.Action == "pause" {
		stateStr = "paused"
	} else if req.Action == "stop" {
		stateStr = "stopped"
	}

	state := &domain.PlaybackState{
		State:          stateStr,
		CurrentTrackID: req.TrackID,
		PositionMS:     req.PositionMS,
		UpdatedAt:      nowMS,
		Title:          req.Title,
		Artist:         req.Artist,
		ThumbnailURL:   req.ThumbnailURL,
		DurationMS:     req.DurationMS,
		SourceURL:      req.SourceURL,
	}

	if err := u.redisRepo.SavePlaybackState(ctx, roomID, state); err != nil {
		return nil, err
	}

	return state, nil
}
