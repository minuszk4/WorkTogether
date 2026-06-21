package ws

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
	"github.com/worktogether/services/collab-service/internal/domain"
	"github.com/worktogether/services/collab-service/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
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
	Usecase    *usecase.CollabUsecase
	jwtSecret  string
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *WSMessage
	mutex      sync.RWMutex
}

func NewHub(jwtSecret string, uc *usecase.CollabUsecase) *Hub {
	return &Hub{
		Usecase:    uc,
		jwtSecret:  jwtSecret,
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WSMessage),
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
			log.Printf("Client %s kết nối vào phòng collab %s\n", client.UserID, client.RoomID)

			// Gửi ngay trạng thái note hiện tại cho client mới kết nối
			ctx := context.Background()
			note, blocks, err := h.Usecase.GetOrCreateNote(ctx, client.RoomID)
			if err == nil {
				syncPayload := map[string]interface{}{
					"note":   note,
					"blocks": blocks,
				}
				payloadBytes, _ := json.Marshal(syncPayload)
				msg := WSMessage{
					Event:   "note:sync",
					RoomID:  client.RoomID,
					Payload: json.RawMessage(payloadBytes),
				}
				msgBytes, _ := json.Marshal(msg)
				client.Send <- msgBytes
			} else {
				log.Printf("Lỗi lấy/tạo note cho phòng %s: %v\n", client.RoomID, err)
			}

		case client := <-h.unregister:
			h.mutex.Lock()
			if rooms, exists := h.rooms[client.RoomID]; exists {
				if _, ok := rooms[client]; ok {
					delete(rooms, client)
					close(client.Send)
					log.Printf("Client %s rời phòng collab %s\n", client.UserID, client.RoomID)
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
		noteID := c.RoomID // We mapped NoteID = RoomID for simplicity

		switch rawMsg.Event {
		case "block:add":
			var b domain.NoteBlock
			if err := json.Unmarshal(rawMsg.Payload, &b); err == nil {
				b.UpdatedBy = c.UserID
				err = c.Hub.Usecase.AddBlock(ctx, noteID, &b)
				if err == nil {
					c.Hub.BroadcastToRoom(c.RoomID, "block:created", b)
				} else {
					log.Printf("Lỗi AddBlock: %v\n", err)
				}
			}

		case "block:update":
			var b domain.NoteBlock
			if err := json.Unmarshal(rawMsg.Payload, &b); err == nil {
				b.UpdatedBy = c.UserID
				err = c.Hub.Usecase.UpdateBlock(ctx, noteID, &b)
				if err == nil {
					c.Hub.BroadcastToRoom(c.RoomID, "block:updated", b)
				} else {
					log.Printf("Lỗi UpdateBlock: %v\n", err)
				}
			}

		case "block:delete":
			var req struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &req); err == nil {
				err = c.Hub.Usecase.DeleteBlock(ctx, noteID, req.ID)
				if err == nil {
					c.Hub.BroadcastToRoom(c.RoomID, "block:deleted", map[string]string{"id": req.ID})
				} else {
					log.Printf("Lỗi DeleteBlock: %v\n", err)
				}
			}

		case "block:move":
			var req struct {
				BlockIDs []string `json:"block_ids"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &req); err == nil {
				err = c.Hub.Usecase.UpdateBlocksOrder(ctx, noteID, req.BlockIDs)
				if err == nil {
					c.Hub.BroadcastToRoom(c.RoomID, "block:ordered", map[string]interface{}{"block_ids": req.BlockIDs})
				} else {
					log.Printf("Lỗi UpdateBlocksOrder: %v\n", err)
				}
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

func ServeCollabWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		tokenStr := c.Query("token")

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực"})
			return
		}

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
			log.Printf("Lỗi nâng cấp WebSocket collab: %v\n", err)
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
