package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	delivery "github.com/worktogether/services/chat-service/internal/delivery/http"
)

type EventWorker struct {
	rdb *redis.Client
	hub *delivery.Hub
}

func NewEventWorker(rdb *redis.Client, hub *delivery.Hub) *EventWorker {
	return &EventWorker{
		rdb: rdb,
		hub: hub,
	}
}

type RoomEventPayload struct {
	Event  string `json:"event"`
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

func (w *EventWorker) Start(ctx context.Context) {
	log.Println("Redis Stream Worker đang lắng nghe stream:room_events...")

	streamKey := "stream:room_events"

	// Đảm bảo stream tồn tại bằng cách tạo rỗng hoặc đọc từ đầu
	// Nếu chưa có message nào trong stream, XRead có thể trả về lỗi. 
	// Chúng ta sẽ bỏ qua lỗi đó và thử lại.
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Đọc block 0 (chờ vô tận cho đến khi có event)
			streams, err := w.rdb.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamKey, "$"},
				Block:   0,
			}).Result()

			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					payloadStr, ok := msg.Values["payload"].(string)
					if !ok {
						continue
					}

					var payload RoomEventPayload
					if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
						continue
					}

					if payload.Event == "member_banned" || payload.Event == "member_kicked" {
						log.Printf("[Worker] Phát hiện kick/ban: Room=%s, User=%s. Thực hiện ngắt socket...\n", payload.RoomID, payload.UserID)
						w.disconnectUser(payload.RoomID, payload.UserID)
					}
				}
			}
		}
	}
}

func (w *EventWorker) disconnectUser(roomID, userID string) {
	w.hub.Lock()
	defer w.hub.Unlock()

	if clients, ok := w.hub.Rooms[roomID]; ok {
		for client := range clients {
			if client.UserID == userID {
				log.Printf("[Worker] Tìm thấy socket hoạt động của user %s trong room %s. Tiến hành ngắt kết nối.\n", userID, roomID)
				
				delete(w.hub.Rooms[roomID], client)
				close(client.Send)
				client.Conn.Close()
				
				if len(w.hub.Rooms[roomID]) == 0 {
					delete(w.hub.Rooms, roomID)
				}
				break
			}
		}
	}
}
