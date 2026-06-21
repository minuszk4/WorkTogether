package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	roomv1 "github.com/worktogether/services/chat-service/api/v1"
	"github.com/worktogether/services/chat-service/internal/domain"
	"github.com/worktogether/services/chat-service/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS được quản lý bởi Gateway Nginx
	},
}

type ChatHandler struct {
	usecase    *usecase.ChatUsecase
	hub        *Hub
	roomClient roomv1.RoomInternalServiceClient
}

func NewChatHandler(uc *usecase.ChatUsecase, hub *Hub, rc roomv1.RoomInternalServiceClient) *ChatHandler {
	return &ChatHandler{
		usecase:    uc,
		hub:        hub,
		roomClient: rc,
	}
}

// 1. WebSocket Handler
func (h *ChatHandler) HandleWS(c *gin.Context) {
	roomID := c.Param("id")
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu thông tin đăng nhập"})
		return
	}
	userID := userIDVal.(string)

	// Call gRPC room-service để xác thực quyền thành viên
	res, err := h.roomClient.VerifyRoomMember(context.Background(), &roomv1.VerifyRoomMemberRequest{
		RoomID: roomID,
		UserID: userID,
	})
	if err != nil || res == nil || !res.IsMember {
		log.Printf("[WS ERR] Từ chối kết nối WS: User %s không phải thành viên Room %s (Err: %v)\n", userID, roomID, err)
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Bạn không phải thành viên của phòng này."})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS ERR] Lỗi upgrade socket: %v\n", err)
		return
	}

	// Trích xuất permissions lưu vào phiên kết nối (cho phép canModerate và canChat)
	canModerate := false
	canChat := false
	for _, p := range res.Permissions {
		if p == "CAN_MODERATE_MEMBERS" {
			canModerate = true
		}
		if p == "CAN_CHAT" {
			canChat = true
		}
	}

	client := &Client{
		UserID:   userID,
		RoomID:   roomID,
		Username: "User_" + userID[:8], // default
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
		CanChat:  canChat,
	}

	h.hub.Register(client)

	// Chạy ghi/đọc song song
	go client.writePump()
	go client.readPump(h.usecase, canModerate)
}

// 2. REST API: Lấy lịch sử chat
func (h *ChatHandler) GetMessages(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("userID")

	res, err := h.roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
		RoomID: roomID,
		UserID: userID,
	})
	if err != nil || res == nil || !res.IsMember {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Bạn không phải thành viên của phòng này.",
			},
		})
		return
	}

	beforeID := c.Query("before_id")
	limit := 50 // default

	list, err := h.usecase.GetMessagesByRoom(c.Request.Context(), roomID, beforeID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_MESSAGES_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
		"error":   nil,
	})
}

// 3. REST API: Tìm kiếm tin nhắn
func (h *ChatHandler) SearchMessages(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("userID")

	res, err := h.roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
		RoomID: roomID,
		UserID: userID,
	})
	if err != nil || res == nil || !res.IsMember {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Bạn không phải thành viên của phòng này.",
			},
		})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Query parameter q is required",
			},
		})
		return
	}
	list, err := h.usecase.SearchMessages(c.Request.Context(), roomID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SEARCH_MESSAGES_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
		"error":   nil,
	})
}

// WebSocket client pumps implementation
func (c *Client) readPump(uc *usecase.ChatUsecase, canModerate bool) {
	defer func() {
		c.Hub.Unregister(c)
	}()

	c.Conn.SetReadLimit(4096)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming domain.WSMessage
		if err := json.Unmarshal(message, &incoming); err != nil {
			continue
		}

		incoming.RoomID = c.RoomID
		incoming.UserID = c.UserID

		// Xử lý các loại Event khác nhau
		switch incoming.Event {
		case "presence:state_change":
			var payload struct {
				IsPlaying  bool `json:"is_playing"`
				PositionMS int  `json:"position_ms"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)

			c.Hub.UpdateClientPresence(c.RoomID, c.UserID, payload.IsPlaying, payload.PositionMS)

			c.Hub.RLock()
			presence := c.Hub.RoomPresences[c.RoomID]
			c.Hub.RUnlock()

			if presence != nil {
				presence.RLock()
				broadcastMsg := domain.WSMessage{
					Event:  "presence:listener_states",
					RoomID: c.RoomID,
					Payload: gin.H{
						"user_states": presence.MemberStates,
					},
				}
				presence.RUnlock()
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "presence:send_reaction":
			var payload struct {
				Emoji string `json:"emoji"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)

			c.Hub.RecordReaction(c.RoomID, payload.Emoji)

			broadcastMsg := domain.WSMessage{
				Event:  "presence:reaction_broadcast",
				RoomID: c.RoomID,
				Payload: gin.H{
					"user_id": c.UserID,
					"emoji":   payload.Emoji,
				},
			}
			data, _ := json.Marshal(broadcastMsg)
			c.Hub.BroadcastToRoom(c.RoomID, data)

		case "chat:send_message":
			if !c.CanChat {
				log.Printf("[WS WARN] Muted user %s tried to send message in room %s\n", c.UserID, c.RoomID)
				continue
			}
			var payload domain.SendMessagePayload
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)

			msg, err := uc.SaveMessage(context.Background(), c.UserID, c.RoomID, &payload)
			if err == nil && msg != nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:message_received",
					RoomID: c.RoomID,
					Payload: gin.H{
						"id":          msg.ID,
						"sender_id":   msg.SenderID,
						"content":     msg.Content,
						"reply_to_id": msg.ReplyToID,
						"created_at":  msg.CreatedAt,
						"mentions":    payload.Mentions,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "chat:typing":
			if !c.CanChat {
				continue
			}
			var payload domain.TypingPayload
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)

			broadcastMsg := domain.WSMessage{
				Event:  "chat:member_typing",
				RoomID: c.RoomID,
				Payload: gin.H{
					"user_id":   c.UserID,
					"username":  c.Username,
					"is_typing": payload.IsTyping,
				},
			}
			data, _ := json.Marshal(broadcastMsg)
			c.Hub.BroadcastToRoom(c.RoomID, data)

		case "chat:react":
			var payload domain.ReactMessagePayload
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)

			var err error
			if payload.Action == "add" {
				_, err = uc.AddReaction(context.Background(), c.UserID, c.RoomID, payload.MessageID, payload.Emoji)
			} else {
				err = uc.RemoveReaction(context.Background(), c.UserID, c.RoomID, payload.MessageID, payload.Emoji)
			}

			if err == nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:reaction_updated",
					RoomID: c.RoomID,
					Payload: gin.H{
						"message_id": payload.MessageID,
						"user_id":    c.UserID,
						"emoji":      payload.Emoji,
						"action":     payload.Action,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "chat:edit_message":
			var payload struct {
				MessageID string `json:"message_id"`
				Content   string `json:"content"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)
			msg, err := uc.EditMessage(context.Background(), c.UserID, c.RoomID, payload.MessageID, payload.Content)
			if err == nil && msg != nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:message_edited",
					RoomID: c.RoomID,
					Payload: gin.H{
						"message_id": msg.ID,
						"content":    msg.Content,
						"is_edited":  true,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "chat:delete_message":
			var payload struct {
				MessageID string `json:"message_id"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)
			err := uc.DeleteMessage(context.Background(), c.UserID, c.RoomID, payload.MessageID, canModerate)
			if err == nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:message_deleted",
					RoomID: c.RoomID,
					Payload: gin.H{
						"message_id": payload.MessageID,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "chat:pin_message":
			var payload struct {
				MessageID string `json:"message_id"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)
			pin, err := uc.PinMessage(context.Background(), c.UserID, c.RoomID, payload.MessageID)
			if err == nil && pin != nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:message_pinned",
					RoomID: c.RoomID,
					Payload: gin.H{
						"message_id": pin.MessageID,
						"pinned_by":  pin.PinnedBy,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}

		case "chat:unpin_message":
			var payload struct {
				MessageID string `json:"message_id"`
			}
			payloadBytes, _ := json.Marshal(incoming.Payload)
			_ = json.Unmarshal(payloadBytes, &payload)
			err := uc.UnpinMessage(context.Background(), c.UserID, c.RoomID, payload.MessageID)
			if err == nil {
				broadcastMsg := domain.WSMessage{
					Event:  "chat:message_unpinned",
					RoomID: c.RoomID,
					Payload: gin.H{
						"message_id": payload.MessageID,
					},
				}
				data, _ := json.Marshal(broadcastMsg)
				c.Hub.BroadcastToRoom(c.RoomID, data)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
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

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
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
