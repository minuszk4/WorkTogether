package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/notification-service/internal/domain"
	"github.com/worktogether/services/notification-service/internal/usecase"
)

type RedisWorker struct {
	rdb      *redis.Client
	usecase  *usecase.NotificationUsecase
	stream   string
	group    string
	consumer string
}

func NewRedisWorker(rdb *redis.Client, u *usecase.NotificationUsecase, stream string) *RedisWorker {
	worker := &RedisWorker{
		rdb:      rdb,
		usecase:  u,
		stream:   stream,
		group:    "notification-group",
		consumer: "notification-consumer-1",
	}

	go worker.startListening()
	return worker
}

func (w *RedisWorker) startListening() {
	ctx := context.Background()

	// Tạo Consumer Group nếu chưa tồn tại
	err := w.rdb.XGroupCreateMkStream(ctx, w.stream, w.group, "$").Err()
	if err != nil {
		// Log và bỏ qua nếu group đã tồn tại
		log.Printf("XGroupCreate info: %v\n", err)
	}

	log.Printf("Bắt đầu lắng nghe sự kiện trên Redis Stream: %s\n", w.stream)

	for {
		streams, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.group,
			Consumer: w.consumer,
			Streams:  []string{w.stream, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err != redis.Nil {
				log.Printf("Lỗi đọc Redis Stream: %v. Thử lại sau 2 giây...\n", err)
				time.Sleep(2 * time.Second)
			}
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				w.processMessage(ctx, message)
			}
		}
	}
}

func (w *RedisWorker) processMessage(ctx context.Context, message redis.XMessage) {
	// Giải mã payload từ Redis Stream message
	payloadStr, exists := message.Values["payload"].(string)
	if !exists {
		log.Printf("Bỏ qua message không hợp lệ, thiếu payload field: %s\n", message.ID)
		w.rdb.XAck(ctx, w.stream, w.group, message.ID)
		return
	}

	var payload struct {
		ReceiverID string `json:"receiver_id"`
		SenderID   string `json:"sender_id"`
		Type       string `json:"type"`
		Content    string `json:"content"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Printf("Lỗi giải mã JSON payload: %v\n", err)
		w.rdb.XAck(ctx, w.stream, w.group, message.ID)
		return
	}

	n := &domain.Notification{
		ReceiverID: payload.ReceiverID,
		SenderID:   payload.SenderID,
		Type:       payload.Type,
		Content:    payload.Content,
		IsRead:     false,
		CreatedAt:  time.Now(),
	}

	err := w.usecase.SendNotification(ctx, n)
	if err != nil {
		log.Printf("Lỗi gửi/lưu thông báo: %v\n", err)
		// Vẫn ACK để tránh nghẽn hàng đợi trong trường hợp lỗi DB
		w.rdb.XAck(ctx, w.stream, w.group, message.ID)
		return
	}

	// Xác nhận xử lý thành công (Acknowledge)
	w.rdb.XAck(ctx, w.stream, w.group, message.ID)
}
