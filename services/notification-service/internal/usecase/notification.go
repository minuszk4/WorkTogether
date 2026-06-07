package usecase

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/worktogether/services/notification-service/internal/domain"
	"github.com/worktogether/services/notification-service/internal/repository"
)

type NotificationUsecase struct {
	postgresRepo  *repository.PostgresRepository
	activeStreams map[string]chan *domain.Notification
	mutex         sync.RWMutex
}

func NewNotificationUsecase(pg *repository.PostgresRepository) *NotificationUsecase {
	return &NotificationUsecase{
		postgresRepo:  pg,
		activeStreams: make(map[string]chan *domain.Notification),
	}
}

func (u *NotificationUsecase) RegisterStream(userID string) chan *domain.Notification {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// Nếu stream cũ tồn tại, đóng nó
	if ch, ok := u.activeStreams[userID]; ok {
		close(ch)
	}

	ch := make(chan *domain.Notification, 50)
	u.activeStreams[userID] = ch
	log.Printf("SSE client registered for user: %s\n", userID)
	return ch
}

func (u *NotificationUsecase) UnregisterStream(userID string, ch chan *domain.Notification) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if currentCh, ok := u.activeStreams[userID]; ok && currentCh == ch {
		delete(u.activeStreams, userID)
		close(ch)
		log.Printf("SSE client unregistered for user: %s\n", userID)
	}
}

func (u *NotificationUsecase) SendNotification(ctx context.Context, n *domain.Notification) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	n.IsRead = false

	// 1. Lưu vào Database Postgres
	if err := u.postgresRepo.SaveNotification(ctx, n); err != nil {
		return err
	}

	// 2. Đẩy qua SSE Stream nếu người dùng đang Online (có active stream)
	u.mutex.RLock()
	ch, ok := u.activeStreams[n.ReceiverID]
	u.mutex.RUnlock()

	if ok {
		select {
		case ch <- n:
			log.Printf("Đã đẩy thông báo tới SSE cho user %s thực tế thành công\n", n.ReceiverID)
		default:
			log.Printf("Hàng đợi SSE đầy cho user %s, bỏ qua đẩy realtime\n", n.ReceiverID)
		}
	}

	return nil
}

func (u *NotificationUsecase) GetNotifications(ctx context.Context, userID string) ([]*domain.Notification, error) {
	return u.postgresRepo.GetUserNotifications(ctx, userID, 50)
}

func (u *NotificationUsecase) MarkRead(ctx context.Context, id, userID string) error {
	return u.postgresRepo.MarkAsRead(ctx, id, userID)
}

func (u *NotificationUsecase) MarkAllRead(ctx context.Context, userID string) error {
	return u.postgresRepo.MarkAllAsRead(ctx, userID)
}

func (u *NotificationUsecase) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return u.postgresRepo.GetUnreadCount(ctx, userID)
}
