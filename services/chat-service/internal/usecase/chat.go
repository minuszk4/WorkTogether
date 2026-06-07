package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/worktogether/services/chat-service/internal/domain"
	"github.com/worktogether/services/chat-service/internal/repository"
)

var (
	ErrMessageNotFound = errors.New("không tìm thấy tin nhắn")
	ErrUnauthorized    = errors.New("bạn không có quyền thực hiện hành động này")
	ErrEditTimeExpired = errors.New("chỉ được sửa tin nhắn trong vòng 15 phút sau khi gửi")
)

type ChatUsecase struct {
	repo *repository.PostgresRepository
}

func NewChatUsecase(repo *repository.PostgresRepository) *ChatUsecase {
	return &ChatUsecase{repo: repo}
}

func (u *ChatUsecase) SaveMessage(ctx context.Context, senderID, roomID string, req *domain.SendMessagePayload) (*domain.Message, error) {
	msg := &domain.Message{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   req.Content,
		ReplyToID: req.ReplyToID,
		IsEdited:  false,
		CreatedAt: time.Now(),
	}

	if err := u.repo.SaveMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (u *ChatUsecase) GetMessagesByRoom(ctx context.Context, roomID string, beforeID string, limit int) ([]*domain.Message, error) {
	return u.repo.GetMessagesByRoom(ctx, roomID, beforeID, limit)
}

func (u *ChatUsecase) EditMessage(ctx context.Context, userID, msgID, content string) (*domain.Message, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrMessageNotFound
	}

	// Chỉ người gửi mới được sửa tin
	if msg.SenderID != userID {
		return nil, ErrUnauthorized
	}

	// Kiểm tra 15 phút
	if time.Since(msg.CreatedAt) > 15*time.Minute {
		return nil, ErrEditTimeExpired
	}

	if err := u.repo.UpdateMessageContent(ctx, msgID, content); err != nil {
		return nil, err
	}

	msg.Content = content
	msg.IsEdited = true
	return msg, nil
}

func (u *ChatUsecase) DeleteMessage(ctx context.Context, userID, msgID string, canModerate bool) error {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrMessageNotFound
	}

	// Người gửi có quyền xóa tin của mình, hoặc moderator có quyền xóa bất kỳ tin nào
	if msg.SenderID != userID && !canModerate {
		return ErrUnauthorized
	}

	return u.repo.DeleteMessage(ctx, msgID)
}

func (u *ChatUsecase) AddReaction(ctx context.Context, userID, msgID, emoji string) (*domain.MessageReaction, error) {
	rx := &domain.MessageReaction{
		MessageID: msgID,
		UserID:    userID,
		Emoji:     emoji,
	}

	if err := u.repo.AddReaction(ctx, rx); err != nil {
		return nil, err
	}
	return rx, nil
}

func (u *ChatUsecase) RemoveReaction(ctx context.Context, userID, msgID, emoji string) error {
	return u.repo.RemoveReaction(ctx, msgID, userID, emoji)
}

func (u *ChatUsecase) PinMessage(ctx context.Context, userID, roomID, msgID string) (*domain.MessagePin, error) {
	pin := &domain.MessagePin{
		MessageID: msgID,
		RoomID:    roomID,
		PinnedBy:  userID,
	}

	if err := u.repo.PinMessage(ctx, pin); err != nil {
		return nil, err
	}
	return pin, nil
}

func (u *ChatUsecase) UnpinMessage(ctx context.Context, msgID string) error {
	return u.repo.UnpinMessage(ctx, msgID)
}
