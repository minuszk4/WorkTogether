package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	roomv1 "github.com/worktogether/services/timer-service/api/v1"
	"github.com/worktogether/services/timer-service/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is managed at Nginx Gateway
	},
}

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	RoomID string
	UserID string
}

type WSMessage struct {
	Event   string          `json:"event"`
	RoomID  string          `json:"room_id"`
	Payload json.RawMessage `json:"payload"`
}

type Hub struct {
	Usecase    *usecase.TimerUsecase
	jwtSecret  string
	roomClient roomv1.RoomInternalServiceClient
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *WSMessage
	mutex      sync.RWMutex
}

func NewHub(jwtSecret string, roomClient roomv1.RoomInternalServiceClient) *Hub {
	return &Hub{
		jwtSecret:  jwtSecret,
		roomClient: roomClient,
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WSMessage),
	}
}

func (h *Hub) canControlTimer(ctx context.Context, roomID, userID string) bool {
	if h.roomClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	member, err := h.roomClient.VerifyRoomMember(ctx, &roomv1.VerifyRoomMemberRequest{RoomID: roomID, UserID: userID})
	if err != nil || member == nil || !member.IsMember {
		return false
	}

	for _, permission := range member.Permissions {
		if strings.EqualFold(permission, "CAN_CONTROL_PLAYBACK") {
			return true
		}
	}
	return false
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
			log.Printf("Client %s kết nối vào phòng timer %s\n", client.UserID, client.RoomID)

			// Gửi ngay trạng thái timer hiện tại cho client mới kết nối
			ctx := context.Background()
			state, err := h.Usecase.GetState(ctx, client.RoomID)
			if err == nil {
				payloadBytes, _ := json.Marshal(state)
				msg := WSMessage{
					Event:   "timer:sync",
					RoomID:  client.RoomID,
					Payload: json.RawMessage(payloadBytes),
				}
				msgBytes, _ := json.Marshal(msg)
				client.Send <- msgBytes
			}

			// Đăng ký ticker nếu cần
			go h.Usecase.RegisterTickerIfNeeded(ctx, client.RoomID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if rooms, exists := h.rooms[client.RoomID]; exists {
				if _, ok := rooms[client]; ok {
					delete(rooms, client)
					close(client.Send)
					log.Printf("Client %s rời phòng timer %s\n", client.UserID, client.RoomID)
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

func (h *Hub) BroadcastToRoom(roomID string, event string, payload interface{}) {
	payloadBytes, _ := json.Marshal(payload)
	h.broadcast <- &WSMessage{
		Event:   event,
		RoomID:  roomID,
		Payload: json.RawMessage(payloadBytes),
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

		var rawMsg struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(messageBytes, &rawMsg); err != nil {
			continue
		}

		ctx := context.Background()
		if !c.Hub.canControlTimer(ctx, c.RoomID, c.UserID) {
			log.Printf("User %s không có quyền điều khiển timer phòng %s", c.UserID, c.RoomID)
			continue
		}

		switch rawMsg.Event {
		case "timer:start":
			var req struct {
				DurationSeconds int `json:"duration_seconds"`
				BreakSeconds    int `json:"break_seconds"`
				Cycles          int `json:"cycles"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &req); err == nil {
				_ = c.Hub.Usecase.StartTimer(ctx, c.RoomID, req.DurationSeconds, req.BreakSeconds, req.Cycles)
			}
		case "timer:pause":
			_ = c.Hub.Usecase.PauseTimer(ctx, c.RoomID)
		case "timer:resume":
			_ = c.Hub.Usecase.ResumeTimer(ctx, c.RoomID)
		case "timer:stop":
			_ = c.Hub.Usecase.StopTimer(ctx, c.RoomID)
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

func ServeTimerWS(hub *Hub) gin.HandlerFunc {
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
			log.Printf("Lỗi nâng cấp WebSocket timer: %v\n", err)
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
