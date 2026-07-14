package usecase

import (
	"context"

	"github.com/worktogether/services/collab-service/internal/domain"
	"github.com/worktogether/services/collab-service/internal/repository"
)

type CollabUsecase struct {
	repo *repository.PostgresRepository
}

func NewCollabUsecase(repo *repository.PostgresRepository) *CollabUsecase {
	return &CollabUsecase{repo: repo}
}

func (uc *CollabUsecase) GetOrCreateNote(ctx context.Context, roomID string) (*domain.Note, []*domain.NoteBlock, error) {
	return uc.repo.GetOrCreateNote(ctx, roomID)
}

func (uc *CollabUsecase) AddBlock(ctx context.Context, noteID string, b *domain.NoteBlock) error {
	return uc.repo.AddBlock(ctx, noteID, b)
}

func (uc *CollabUsecase) UpdateBlock(ctx context.Context, noteID string, b *domain.NoteBlock) error {
	return uc.repo.UpdateBlock(ctx, noteID, b)
}

func (uc *CollabUsecase) DeleteBlock(ctx context.Context, noteID string, blockID string) error {
	return uc.repo.DeleteBlock(ctx, noteID, blockID)
}

func (uc *CollabUsecase) UpdateBlocksOrder(ctx context.Context, noteID string, blockIDs []string) error {
	return uc.repo.UpdateBlocksOrder(ctx, noteID, blockIDs)
}

func (uc *CollabUsecase) GetWhiteboardSnapshot(ctx context.Context, roomID string) (*domain.WhiteboardSnapshot, error) {
	return uc.repo.GetWhiteboardSnapshot(ctx, roomID)
}

func (uc *CollabUsecase) SaveWhiteboardSnapshot(ctx context.Context, roomID string, snapshot []byte) error {
	return uc.repo.SaveWhiteboardSnapshot(ctx, roomID, snapshot)
}
