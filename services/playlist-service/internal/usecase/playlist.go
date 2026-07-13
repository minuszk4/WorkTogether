package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/worktogether/services/playlist-service/internal/domain"
	"github.com/worktogether/services/playlist-service/internal/repository"
)

type PlaylistUsecase struct {
	postgresRepo *repository.PostgresRepository
}

func NewPlaylistUsecase(pg *repository.PostgresRepository) *PlaylistUsecase {
	return &PlaylistUsecase{postgresRepo: pg}
}

func (u *PlaylistUsecase) CreatePlaylist(ctx context.Context, name string, roomID, userID *string) (*domain.Playlist, error) {
	p := &domain.Playlist{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
	}

	if err := u.postgresRepo.CreatePlaylist(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *PlaylistUsecase) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return u.postgresRepo.GetPlaylistByID(ctx, id)
}

func (u *PlaylistUsecase) GetPlaylistByTrackID(ctx context.Context, trackID string) (*domain.Playlist, error) {
	return u.postgresRepo.GetPlaylistByTrackID(ctx, trackID)
}

func (u *PlaylistUsecase) GetRoomPlaylists(ctx context.Context, roomID string) ([]*domain.Playlist, error) {
	return u.postgresRepo.GetRoomPlaylists(ctx, roomID)
}

func (u *PlaylistUsecase) GetUserPlaylists(ctx context.Context, userID string) ([]*domain.Playlist, error) {
	return u.postgresRepo.GetUserPlaylists(ctx, userID)
}

func (u *PlaylistUsecase) DeletePlaylist(ctx context.Context, id string) error {
	return u.postgresRepo.DeletePlaylist(ctx, id)
}

func (u *PlaylistUsecase) AddTrack(ctx context.Context, playlistID string, req *domain.AddTrackRequest, userID string) (*domain.PlaylistTrack, error) {
	t := &domain.PlaylistTrack{
		ID:         uuid.New().String(),
		PlaylistID: playlistID,
		TrackID:    req.TrackID,
		Title:      req.Title,
		Artist:     req.Artist,
		Thumbnail:  req.ThumbnailURL,
		DurationMS: req.DurationMS,
		SourceURL:  req.SourceURL,
		AddedBy:    userID,
		CreatedAt:  time.Now(),
	}

	if err := u.postgresRepo.AddTrack(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (u *PlaylistUsecase) RemoveTrack(ctx context.Context, playlistID, trackItemID string) error {
	return u.postgresRepo.RemoveTrack(ctx, playlistID, trackItemID)
}

func (u *PlaylistUsecase) MoveTrack(ctx context.Context, playlistID, trackItemID string, newPos int) error {
	return u.postgresRepo.MoveTrack(ctx, playlistID, trackItemID, newPos)
}

func (u *PlaylistUsecase) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*domain.PlaylistTrack, error) {
	return u.postgresRepo.GetPlaylistTracks(ctx, playlistID)
}

func (u *PlaylistUsecase) VoteTrack(ctx context.Context, trackItemID, userID, voteType string) error {
	if voteType == "none" {
		return u.postgresRepo.DeleteVote(ctx, trackItemID, userID)
	}

	v := &domain.PlaylistVote{
		PlaylistTrackID: trackItemID,
		UserID:          userID,
		VoteType:        voteType,
		CreatedAt:       time.Now(),
	}
	return u.postgresRepo.SaveVote(ctx, v)
}
