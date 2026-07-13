package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	rdb  *redis.Client
}

func NewChatUsecase(repo *repository.PostgresRepository, rdb *redis.Client) *ChatUsecase {
	return &ChatUsecase{repo: repo, rdb: rdb}
}

func (u *ChatUsecase) SaveMessage(ctx context.Context, senderID, roomID string, req *domain.SendMessagePayload) (*domain.Message, error) {
	// Filter and deduplicate mentions: skip empty strings, deduplicate list, and filter out senderID
	var uniqueMentions []string
	seen := make(map[string]bool)
	for _, m := range req.Mentions {
		if m != "" && m != senderID && !seen[m] {
			seen[m] = true
			uniqueMentions = append(uniqueMentions, m)
		}
	}
	req.Mentions = uniqueMentions

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

	// Publish mentions to Redis stream
	if u.rdb != nil {
		for _, mentionedUserID := range req.Mentions {
			eventPayload := map[string]interface{}{
				"receiver_id": mentionedUserID,
				"sender_id":   senderID,
				"type":        "mention",
				"content":     "bạn được nhắc đến trong phòng.",
			}
			payloadBytes, err := json.Marshal(eventPayload)
			if err == nil {
				_ = u.rdb.XAdd(ctx, &redis.XAddArgs{
					Stream: "stream:notification_trigger",
					Values: map[string]interface{}{
						"payload": string(payloadBytes),
					},
				}).Err()
			}
		}
	}
	return msg, nil
}

func (u *ChatUsecase) GetMessagesByRoom(ctx context.Context, roomID string, beforeID string, limit int) ([]*domain.Message, error) {
	return u.repo.GetMessagesByRoom(ctx, roomID, beforeID, limit)
}

func (u *ChatUsecase) EditMessage(ctx context.Context, userID, roomID, msgID, content string) (*domain.Message, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrMessageNotFound
	}
	if msg.RoomID != roomID {
		return nil, ErrUnauthorized
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

func (u *ChatUsecase) DeleteMessage(ctx context.Context, userID, roomID, msgID string, canModerate bool) error {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrMessageNotFound
	}
	if msg.RoomID != roomID {
		return ErrUnauthorized
	}

	// Người gửi có quyền xóa tin của mình, hoặc moderator có quyền xóa bất kỳ tin nào
	if msg.SenderID != userID && !canModerate {
		return ErrUnauthorized
	}

	return u.repo.DeleteMessage(ctx, msgID)
}

func (u *ChatUsecase) AdminDeleteMessage(ctx context.Context, roomID, msgID string) (*domain.Message, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrMessageNotFound
	}
	if msg.RoomID != roomID {
		return nil, ErrUnauthorized
	}
	if err := u.repo.DeleteMessage(ctx, msgID); err != nil {
		return nil, err
	}
	return msg, nil
}

func (u *ChatUsecase) AddReaction(ctx context.Context, userID, roomID, msgID, emoji string) (*domain.MessageReaction, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.RoomID != roomID {
		return nil, ErrUnauthorized
	}

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

func (u *ChatUsecase) RemoveReaction(ctx context.Context, userID, roomID, msgID, emoji string) error {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}
	if msg == nil || msg.RoomID != roomID {
		return ErrUnauthorized
	}

	return u.repo.RemoveReaction(ctx, msgID, userID, emoji)
}

func (u *ChatUsecase) PinMessage(ctx context.Context, userID, roomID, msgID string) (*domain.MessagePin, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.RoomID != roomID {
		return nil, ErrUnauthorized
	}

	pinnedCount, err := u.repo.GetPinnedCount(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if pinnedCount >= 10 {
		return nil, errors.New("vượt quá giới hạn 10 tin nhắn ghim cho phòng này")
	}

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

func (u *ChatUsecase) UnpinMessage(ctx context.Context, userID, roomID, msgID string) error {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}
	if msg == nil || msg.RoomID != roomID {
		return ErrUnauthorized
	}

	return u.repo.UnpinMessage(ctx, msgID)
}

func (u *ChatUsecase) SearchMessages(ctx context.Context, userID, roomID, query string) ([]*domain.Message, error) {
	// Defense-in-depth: Log audit cho hành động search
	// Thực tế usecase nên kiểm tra lại quyền nếu không gọi qua handler, nhưng ở đây log lại trước.
	// log.Printf("[AUDIT] User %s is searching messages in room %s with query: %s", userID, roomID, query)
	return u.repo.SearchMessages(ctx, roomID, query)
}
