package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	roomv1 "github.com/worktogether/services/playback-service/api/v1"
	"github.com/worktogether/services/playback-service/internal/domain"
	"github.com/worktogether/services/playback-service/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Cấu hình CORS ở Nginx Gateway
	},
}

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	RoomID string
	UserID string
}

type Hub struct {
	usecase    *usecase.PlaybackUsecase
	jwtSecret  string
	rooms      map[string]map[*Client]bool // map room_id -> list client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *domain.WSMessage
	roomClient roomv1.RoomInternalServiceClient
	mutex      sync.RWMutex
}

func NewHub(u *usecase.PlaybackUsecase, rc roomv1.RoomInternalServiceClient, jwtSecret string) *Hub {
	return &Hub{
		usecase:    u,
		jwtSecret:  jwtSecret,
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *domain.WSMessage),
		roomClient: rc,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			h.mutex.Unlock()
			log.Printf("Client %s kết nối vào phòng playback %s\n", client.UserID, client.RoomID)

			// Gửi ngay trạng thái playback hiện tại cho client mới kết nối
			ctx := context.Background()
			state, err := h.usecase.GetOrCreateState(ctx, client.RoomID)
			if err == nil {
				payloadBytes, _ := json.Marshal(state)
				msg := domain.WSMessage{
					Event:  "playback:sync",
					RoomID: client.RoomID,
					Payload: json.RawMessage(payloadBytes),
				}
				msgBytes, _ := json.Marshal(msg)
				client.Send <- msgBytes
			}

		case client := <-h.unregister:
			h.mutex.Lock()
			if rooms, exists := h.rooms[client.RoomID]; exists {
				if _, ok := rooms[client]; ok {
					delete(rooms, client)
					close(client.Send)
					log.Printf("Client %s rời phòng playback %s\n", client.UserID, client.RoomID)
				}
				if len(rooms) == 0 {
					delete(h.rooms, client.RoomID)
				}
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			clients := h.rooms[message.RoomID]
			if clients != nil {
				msgBytes, err := json.Marshal(message)
				if err == nil {
					for client := range clients {
						select {
						case client.Send <- msgBytes:
						default:
							h.mutex.RUnlock()
							h.mutex.Lock()
							delete(clients, client)
							close(client.Send)
							h.mutex.Unlock()
							h.mutex.RLock()
						}
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Nhận dữ liệu thô và parse Event
		var rawMsg struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(messageBytes, &rawMsg); err != nil {
			continue
		}

		ctx := context.Background()

		// 1. Xử lý đồng bộ NTP-like Time
		if rawMsg.Event == "sync:ping" {
			t2 := time.Now().UnixNano() / int64(time.Millisecond) // Server nhận ping

			var ping domain.PingPayload
			if err := json.Unmarshal(rawMsg.Payload, &ping); err != nil {
				continue
			}

			t3 := time.Now().UnixNano() / int64(time.Millisecond) // Server gửi pong

			pongPayload := domain.PongPayload{
				T1: ping.T1,
				T2: t2,
				T3: t3,
			}
			payloadBytes, _ := json.Marshal(pongPayload)

			resp := domain.WSMessage{
				Event:   "sync:pong",
				RoomID:  c.RoomID,
				Payload: json.RawMessage(payloadBytes),
			}
			respBytes, _ := json.Marshal(resp)
			c.Send <- respBytes
			continue
		}

		// 2. Xử lý lệnh điều khiển Playback
		if rawMsg.Event == "playback:control" {
			var ctrl domain.ControlPayload
			if err := json.Unmarshal(rawMsg.Payload, &ctrl); err != nil {
				continue
			}

			// Kiểm tra xem phòng có Guest DJ đang hoạt động không
			guestDJ, err := c.Hub.usecase.GetGuestDJ(ctx, c.RoomID)
			if err == nil && guestDJ != "" {
				// Nếu người gửi không phải là Guest DJ hiện tại
				if c.UserID != guestDJ {
					// Kiểm tra xem người gửi có phải là Host (OWNER) không
					res, err := c.Hub.roomClient.VerifyRoomMember(ctx, &roomv1.VerifyRoomMemberRequest{
						RoomID: c.RoomID,
						UserID: c.UserID,
					})
					if err != nil || res == nil || res.Role != "OWNER" {
						// Không phải Host cũng không phải Guest DJ -> Trả lỗi và bỏ qua lệnh
						payloadBytes, _ := json.Marshal(map[string]interface{}{
							"code":    "FORBIDDEN",
							"message": "Phòng đang có Guest DJ làm chủ bàn nhạc. Chỉ Host hoặc Guest DJ hiện tại mới có quyền thay đổi phát nhạc.",
						})
						resp := domain.WSMessage{
							Event:   "playback:error",
							RoomID:  c.RoomID,
							Payload: json.RawMessage(payloadBytes),
						}
						respBytes, _ := json.Marshal(resp)
						c.Send <- respBytes
						continue
					}
				}
			}

			// Lưu trạng thái mới vào Redis
			state, err := c.Hub.usecase.UpdateState(ctx, c.RoomID, &ctrl)
			if err != nil {
				log.Printf("Lỗi lưu playback state: %v\n", err)
				continue
			}

			// Broadcast trạng thái mới tới cả phòng
			payloadBytes, _ := json.Marshal(state)
			c.Hub.broadcast <- &domain.WSMessage{
				Event:   "playback:sync",
				RoomID:  c.RoomID,
				Payload: json.RawMessage(payloadBytes),
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Thêm các message hàng đợi
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServePlaybackWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		tokenStr := c.Query("token")

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực"})
			return
		}

		// Validate token
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(hub.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc hết hạn"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["type"] != "access_token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Loại token không hợp lệ"})
			return
		}

		userID, _ := claims["sub"].(string)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Lỗi nâng cấp WebSocket: %v\n", err)
			return
		}

		client := &Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			RoomID: roomID,
			UserID: userID,
		}

		client.Hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}

func (h *Hub) BroadcastToRoom(roomID string, event string, payload interface{}) {
	payloadBytes, _ := json.Marshal(payload)
	h.broadcast <- &domain.WSMessage{
		Event:   event,
		RoomID:  roomID,
		Payload: json.RawMessage(payloadBytes),
	}
}
