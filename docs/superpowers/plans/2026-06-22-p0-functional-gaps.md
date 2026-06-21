# P0 Functional Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Triển khai các tính năng P0 bị thiếu (Forgot/Reset/Change password trong auth-service, Delete/Mute trong room-service, WS Message management + Search + Mention triggers trong chat-service).

**Architecture:** Bổ sung các REST endpoint tương ứng trên các dịch vụ Go, viết migrations cho DB PostgreSQL của room-service, bổ sung switch case xử lý WS event trong chat-service và gửi trigger thông báo qua Redis Stream.

**Tech Stack:** Go (Gin, sqlx, go-redis, go-jwt), PostgreSQL, Redis.

## Global Constraints
- Target Go version: 1.21+
- Tôn trọng nguyên tắc Onion/Clean Architecture trong microservices Go.
- Các API RESTful mới phải trả về định dạng JSON thống nhất của hệ thống: `{"success": true, "data": ...}` hoặc `{"success": false, "error": {"code": "...", "message": "..."}}`.
- Viết test suite đầy đủ cho repository/usecase mới.

---

### Task 1: Room Member Muting Database Migration (`room-service`)

**Files:**
- Create: `services/room-service/db/migrations/000005_add_mute_to_members.up.sql`
- Create: `services/room-service/db/migrations/000005_add_mute_to_members.down.sql`
- Modify: `services/room-service/internal/domain/models.go`
- Modify: `services/room-service/internal/repository/postgres.go`

**Interfaces:**
- Consumes: Postgres Connection DB in room-service.
- Produces: `muted_until` column in `room_members` table and mapped fields in `domain.RoomMember`.

- [ ] **Step 1: Write migration SQL files**
  Create `services/room-service/db/migrations/000005_add_mute_to_members.up.sql` with:
  ```sql
  ALTER TABLE room_members ADD COLUMN IF NOT EXISTS muted_until TIMESTAMP WITH TIME ZONE;
  ```
  Create `services/room-service/db/migrations/000005_add_mute_to_members.down.sql` with:
  ```sql
  ALTER TABLE room_members DROP COLUMN IF EXISTS muted_until;
  ```

- [ ] **Step 2: Update models.go**
  Modify `RoomMember` struct in `services/room-service/internal/domain/models.go` to add `MutedUntil` field:
  ```go
  type RoomMember struct {
      ID              string     `json:"id"`
      RoomID          string     `json:"room_id"`
      UserID          string     `json:"user_id"`
      RoleID          string     `json:"role_id,omitempty"`
      RoleType        string     `json:"role_type"`
      ActiveSubRoomID *string    `json:"active_sub_room_id,omitempty"`
      MutedUntil      *time.Time `json:"muted_until,omitempty"`
      JoinedAt        time.Time  `json:"joined_at"`
  }
  ```

- [ ] **Step 3: Update postgres repository**
  Modify `services/room-service/internal/repository/postgres.go` to scan `muted_until` in `GetMember` (around line 160) and `ListMembers` (around line 180).
  In `GetMember`:
  ```go
  // Modify SELECT query:
  query := `SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members WHERE room_id = $1 AND user_id = $2`
  m := &domain.RoomMember{}
  var roleID sql.NullString
  var activeSubRoomID sql.NullString
  var mutedUntil sql.NullTime
  err := r.db.QueryRowContext(ctx, query, roomID, userID).
      Scan(&m.ID, &m.RoomID, &m.UserID, &roleID, &m.RoleType, &activeSubRoomID, &mutedUntil, &m.JoinedAt)
  // ...
  if mutedUntil.Valid {
      m.MutedUntil = &mutedUntil.Time
  }
  ```
  Also modify `ListMembers` query and scanning in the same manner.
  Add `UpdateMemberMute` method at the end of the file:
  ```go
  func (r *PostgresRepository) UpdateMemberMute(ctx context.Context, roomID, userID string, mutedUntil *time.Time) error {
      query := `UPDATE room_members SET muted_until = $1 WHERE room_id = $2 AND user_id = $3`
      _, err := r.db.ExecContext(ctx, query, mutedUntil, roomID, userID)
      return err
   }
  ```

- [ ] **Step 4: Verify Compilation**
  Run: `go build ./...` inside `services/room-service`
  Expected: SUCCESS

- [ ] **Step 5: Commit**
  ```bash
  git add services/room-service/db/migrations/000005_add_mute_to_members.up.sql services/room-service/db/migrations/000005_add_mute_to_members.down.sql services/room-service/internal/domain/models.go services/room-service/internal/repository/postgres.go
  git commit -m "migration: add muted_until column to room_members"
  ```

---

### Task 2: Room Mute/Unmute & stand-alone Deletion (`room-service`)

**Files:**
- Modify: `services/room-service/internal/usecase/room.go`
- Modify: `services/room-service/internal/delivery/http/handlers.go`
- Modify: `services/room-service/internal/delivery/grpc/server.go`
- Modify: `services/room-service/cmd/server/main.go`

**Interfaces:**
- Consumes: `UpdateMemberMute` from PostgresRepository.
- Produces: Mute/Unmute REST API endpoints, separate room deletion endpoint, and filtered gRPC verify responses.

- [ ] **Step 1: Update room usecase**
  Modify `services/room-service/internal/usecase/room.go` to add `MuteMember`, `UnmuteMember`, and `DeleteRoom` methods:
  ```go
  func (u *RoomUsecase) DeleteRoom(ctx context.Context, userID, roomID string) error {
      member, err := u.repo.GetMember(ctx, roomID, userID)
      if err != nil {
          return err
      }
      if member == nil || member.RoleType != "OWNER" {
          return ErrUnauthorized
      }
      return u.repo.DeleteRoom(ctx, roomID)
  }

  func (u *RoomUsecase) MuteMember(ctx context.Context, requesterID, roomID, targetUserID string, durationSecs int) error {
      reqMember, err := u.repo.GetMember(ctx, roomID, requesterID)
      if err != nil || reqMember == nil || (reqMember.RoleType != "OWNER" && reqMember.RoleType != "MODERATOR") {
          return ErrUnauthorized
      }
      targetMember, err := u.repo.GetMember(ctx, roomID, targetUserID)
      if err != nil {
          return err
      }
      if targetMember == nil {
          return errors.New("thành viên mục tiêu không tồn tại")
      }
      if targetMember.RoleType == "OWNER" {
          return errors.New("không thể mute chủ phòng")
      }
      mutedUntil := time.Now().Add(time.Duration(durationSecs) * time.Second)
      return u.repo.UpdateMemberMute(ctx, roomID, targetUserID, &mutedUntil)
  }

  func (u *RoomUsecase) UnmuteMember(ctx context.Context, requesterID, roomID, targetUserID string) error {
      reqMember, err := u.repo.GetMember(ctx, roomID, requesterID)
      if err != nil || reqMember == nil || (reqMember.RoleType != "OWNER" && reqMember.RoleType != "MODERATOR") {
          return ErrUnauthorized
      }
      targetMember, err := u.repo.GetMember(ctx, roomID, targetUserID)
      if err != nil {
          return err
      }
      if targetMember == nil {
          return errors.New("thành viên mục tiêu không tồn tại")
      }
      return u.repo.UpdateMemberMute(ctx, roomID, targetUserID, nil)
  }
  ```

- [ ] **Step 2: Add REST endpoint controllers**
  Modify `services/room-service/internal/delivery/http/handlers.go`:
  ```go
  func (h *RoomHandler) DeleteRoom(c *gin.Context) {
      roomID := c.Param("id")
      userID := c.GetString("userID")
      if err := h.usecase.DeleteRoom(c.Request.Context(), userID, roomID); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": "Xóa phòng thành công."})
  }

  func (h *RoomHandler) MuteMember(c *gin.Context) {
      roomID := c.Param("id")
      targetUserID := c.Param("user_id")
      requesterID := c.GetString("userID")
      var req domain.MuteRequest
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
          return
      }
      if err := h.usecase.MuteMember(c.Request.Context(), requesterID, roomID, targetUserID, req.DurationSeconds); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": "Mute thành viên thành công."})
  }

  func (h *RoomHandler) UnmuteMember(c *gin.Context) {
      roomID := c.Param("id")
      targetUserID := c.Param("user_id")
      requesterID := c.GetString("userID")
      if err := h.usecase.UnmuteMember(c.Request.Context(), requesterID, roomID, targetUserID); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": "Unmute thành viên thành công."})
  }
  ```

- [ ] **Step 3: Register REST routes**
  Modify `services/room-service/cmd/server/main.go` around line 90:
  ```go
  roomsGroup.DELETE("/:id", handler.DeleteRoom)
  roomsGroup.POST("/:id/members/:user_id/mute", handler.MuteMember)
  roomsGroup.POST("/:id/members/:user_id/unmute", handler.UnmuteMember)
  ```

- [ ] **Step 4: Update gRPC VerifyRoomMember**
  Modify `services/room-service/internal/delivery/grpc/server.go` to filter permissions if the user is currently muted:
  ```go
  // Inside VerifyRoomMember method:
  // After defining permissions...
  if m.MutedUntil != nil && m.MutedUntil.After(time.Now()) {
      filtered := []string{}
      for _, p := range permissions {
          if p != "CAN_CHAT" && p != "CAN_USE_VOICE" {
              filtered = append(filtered, p)
          }
      }
      permissions = filtered
  }
  ```

- [ ] **Step 5: Verify via building and running tests**
  Run: `go test -v ./internal/usecase/...` inside `services/room-service`
  Expected: PASS

- [ ] **Step 6: Commit**
  ```bash
  git add services/room-service/internal/usecase/room.go services/room-service/internal/delivery/http/handlers.go services/room-service/internal/delivery/grpc/server.go services/room-service/cmd/server/main.go
  git commit -m "feat(room): implement mute/unmute and separate room deletion endpoints"
  ```

---

### Task 3: Authentication Forgot/Reset/Change Password (`auth-service`)

**Files:**
- Create: `services/auth-service/pkg/middleware/auth.go`
- Modify: `services/auth-service/internal/domain/models.go`
- Modify: `services/auth-service/internal/repository/postgres.go`
- Modify: `services/auth-service/internal/usecase/email.go`
- Modify: `services/auth-service/internal/usecase/auth.go`
- Modify: `services/auth-service/internal/delivery/http/handlers.go`
- Modify: `services/auth-service/cmd/server/main.go`

**Interfaces:**
- Consumes: SMTP configurations, `UpdatePassword` in PostgresRepository.
- Produces: HTTP Endpoints for `/forgot-password`, `/reset-password`, `/change-password`.

- [ ] **Step 1: Add UpdatePassword in Postgres repository**
  Modify `services/auth-service/internal/repository/postgres.go` to add `UpdatePassword`:
  ```go
  func (r *PostgresRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
      query := `UPDATE accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2`
      _, err := r.db.ExecContext(ctx, query, passwordHash, id)
      return err
  }
  ```

- [ ] **Step 2: Create AuthMiddleware for auth-service**
  Create `services/auth-service/pkg/middleware/auth.go` copying the contents of `services/user-service/pkg/middleware/auth.go` (change package name to `middleware`).

- [ ] **Step 3: Update domain models**
  Modify `services/auth-service/internal/domain/models.go` to add payload structs:
  ```go
  type ForgotPasswordRequest struct {
      Email string `json:"email" binding:"required,email"`
  }

  type ResetPasswordRequest struct {
      Token       string `json:"token" binding:"required"`
      NewPassword string `json:"new_password" binding:"required,min=6"`
  }

  type ChangePasswordRequest struct {
      OldPassword string `json:"old_password" binding:"required"`
      NewPassword string `json:"new_password" binding:"required,min=6"`
  }
  ```

- [ ] **Step 4: Update EmailService**
  Modify `services/auth-service/internal/usecase/email.go` to add `SendPasswordResetEmail`:
  ```go
  func (e *EmailService) SendPasswordResetEmail(toEmail, username, resetURL string) error {
      if !e.IsConfigured() {
          fmt.Printf("[EMAIL-DEV] Gửi link reset mật khẩu tới %s: %s\n", toEmail, resetURL)
          return nil
      }
      subject := "WorkTogether – Khôi phục mật khẩu của bạn"
      body := fmt.Sprintf(`<h2>Xin chào, %s!</h2><p>Nhấp vào liên kết sau để đặt lại mật khẩu: <a href="%s">%s</a></p>`, username, resetURL, resetURL)
      return e.sendHTML(toEmail, subject, body)
  }
  ```

- [ ] **Step 5: Implement password reset logic in AuthUsecase**
  Modify `services/auth-service/internal/usecase/auth.go` to add `ForgotPassword`, `ResetPassword`, and `ChangePassword`:
  ```go
  func (u *AuthUsecase) ForgotPassword(ctx context.Context, email string) error {
      acc, err := u.repo.GetAccountByEmail(ctx, email)
      if err != nil || acc == nil {
          return errors.New("không tìm thấy tài khoản với email này")
      }
      token := jwt.NewWithClaims(jwt.SigningMethodHMAC, jwt.MapClaims{
          "sub":  acc.ID,
          "type": "password_reset",
          "exp":  time.Now().Add(15 * time.Minute).Unix(),
      })
      tokenStr, err := token.SignedString(u.jwtSecret)
      if err != nil {
          return err
      }
      resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", u.appBaseURL, tokenStr)
      return u.emailSvc.SendPasswordResetEmail(acc.Email, acc.Username, resetURL)
  }

  func (u *AuthUsecase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
      token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
          return u.jwtSecret, nil
      })
      if err != nil || !token.Valid {
          return errors.New("token không hợp lệ hoặc đã hết hạn")
      }
      claims, ok := token.Claims.(jwt.MapClaims)
      if !ok || claims["type"] != "password_reset" {
          return errors.New("token không hợp lệ")
      }
      accountID, _ := claims["sub"].(string)
      hashedPassword, err := u.hashPassword(newPassword)
      if err != nil {
          return err
      }
      return u.repo.UpdatePassword(ctx, accountID, hashedPassword)
  }

  func (u *AuthUsecase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
      acc, err := u.repo.GetAccountByID(ctx, userID)
      if err != nil || acc == nil {
          return errors.New("tài khoản không tồn tại")
      }
      if !u.checkPasswordHash(oldPassword, acc.PasswordHash) {
          return errors.New("mật khẩu cũ không chính xác")
      }
      hashedPassword, err := u.hashPassword(newPassword)
      if err != nil {
          return err
      }
      return u.repo.UpdatePassword(ctx, userID, hashedPassword)
  }
  ```

- [ ] **Step 6: Add controllers and routes**
  Modify `services/auth-service/internal/delivery/http/handlers.go` to implement `ForgotPassword`, `ResetPassword`, and `ChangePassword` handlers.
  Register them in `services/auth-service/cmd/server/main.go`:
  ```go
  authGroup.POST("/forgot-password", handler.ForgotPassword)
  authGroup.POST("/reset-password", handler.ResetPassword)
  authGroup.POST("/change-password", middleware.AuthMiddleware(jwtSecret), handler.ChangePassword)
  ```

- [ ] **Step 7: Verify Compilation**
  Run: `go build ./...` inside `services/auth-service`
  Expected: SUCCESS

- [ ] **Step 8: Commit**
  ```bash
  git add services/auth-service/
  git commit -m "feat(auth): implement forgot, reset, and change password features"
  ```

---

### Task 4: WebSocket Message Management & Search Endpoints (`chat-service`)

**Files:**
- Modify: `services/chat-service/internal/repository/postgres.go`
- Modify: `services/chat-service/internal/usecase/chat.go`
- Modify: `services/chat-service/internal/delivery/http/handlers.go`
- Modify: `services/chat-service/cmd/server/main.go`

**Interfaces:**
- Consumes: WebSocket connection events, Postgres connection.
- Produces: WS Events for `chat:message_edited`, `chat:message_deleted`, `chat:message_pinned`, `chat:message_unpinned` and REST endpoint `GET /api/v1/rooms/:id/chat/search`.

- [ ] **Step 1: Add Search query in repository**
  Modify `services/chat-service/internal/repository/postgres.go` to implement `SearchMessages`:
  ```go
  func (r *PostgresRepository) SearchMessages(ctx context.Context, roomID, query string) ([]*domain.Message, error) {
      dbQuery := `
          SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at 
          FROM messages 
          WHERE room_id = $1 AND content ILIKE $2
          ORDER BY created_at DESC 
          LIMIT 50
      `
      rows, err := r.db.QueryContext(ctx, dbQuery, roomID, "%"+query+"%")
      if err != nil {
          return nil, err
      }
      defer rows.Close()
      var list []*domain.Message
      for rows.Next() {
          msg := &domain.Message{}
          var replyTo sql.NullString
          if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Content, &replyTo, &msg.IsEdited, &msg.CreatedAt); err != nil {
              return nil, err
          }
          if replyTo.Valid {
              msg.ReplyToID = replyTo.String
          }
          list = append(list, msg)
      }
      return list, nil
  }
  ```

- [ ] **Step 2: Add Search in Usecase**
  Modify `services/chat-service/internal/usecase/chat.go`:
  ```go
  func (u *ChatUsecase) SearchMessages(ctx context.Context, roomID, query string) ([]*domain.Message, error) {
      return u.repo.SearchMessages(ctx, roomID, query)
  }
  ```

- [ ] **Step 3: Update WS Handler readPump to support edit, delete, pin, unpin**
  Modify `services/chat-service/internal/delivery/http/handlers.go` `readPump` method to handle:
  - `"chat:edit_message"`:
    ```go
    case "chat:edit_message":
        var payload struct {
            MessageID string `json:"message_id"`
            Content   string `json:"content"`
        }
        payloadBytes, _ := json.Marshal(incoming.Payload)
        _ = json.Unmarshal(payloadBytes, &payload)
        msg, err := uc.EditMessage(context.Background(), c.UserID, payload.MessageID, payload.Content)
        if err == nil && msg != nil {
            broadcastMsg := domain.WSMessage{
                Event:  "chat:message_edited",
                RoomID: c.RoomID,
                Payload: gin.H{
                    "message_id": msg.ID,
                    "content":    msg.Content,
                    "is_edited":   true,
                },
            }
            data, _ := json.Marshal(broadcastMsg)
            c.Hub.BroadcastToRoom(c.RoomID, data)
        }
    ```
  - `"chat:delete_message"`:
    ```go
    case "chat:delete_message":
        var payload struct {
            MessageID string `json:"message_id"`
        }
        payloadBytes, _ := json.Marshal(incoming.Payload)
        _ = json.Unmarshal(payloadBytes, &payload)
        err := uc.DeleteMessage(context.Background(), c.UserID, payload.MessageID, canModerate)
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
    ```
  - `"chat:pin_message"`:
    ```go
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
    ```
  - `"chat:unpin_message"`:
    ```go
    case "chat:unpin_message":
        var payload struct {
            MessageID string `json:"message_id"`
        }
        payloadBytes, _ := json.Marshal(incoming.Payload)
        _ = json.Unmarshal(payloadBytes, &payload)
        err := uc.UnpinMessage(context.Background(), payload.MessageID)
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
    ```

- [ ] **Step 4: Create Search REST endpoint**
  Modify `services/chat-service/internal/delivery/http/handlers.go` to add `SearchMessages`:
  ```go
  func (h *ChatHandler) SearchMessages(c *gin.Context) {
      roomID := c.Param("id")
      query := c.Query("q")
      if query == "" {
          c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Query parameter q is required"})
          return
      }
      list, err := h.usecase.SearchMessages(c.Request.Context(), roomID, query)
      if err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
  }
  ```
  Register in `services/chat-service/cmd/server/main.go` inside the `chatGroup` routes block:
  ```go
  chatGroup.GET("/search", handler.SearchMessages)
  ```

- [ ] **Step 5: Verify Compilation**
  Run: `go build ./...` inside `services/chat-service`
  Expected: SUCCESS

- [ ] **Step 6: Commit**
  ```bash
  git add services/chat-service/
  git commit -m "feat(chat): implement edit, delete, pin, and search endpoints"
  ```

---

### Task 5: Mention detection and Redis Notification trigger (`chat-service`)

**Files:**
- Modify: `services/chat-service/internal/domain/models.go`
- Modify: `services/chat-service/internal/usecase/chat.go`
- Modify: `services/chat-service/internal/delivery/http/handlers.go`
- Modify: `services/chat-service/cmd/server/main.go`

**Interfaces:**
- Consumes: Redis Client instance in ChatUsecase, `stream:notification_trigger` stream.
- Produces: Notification trigger events inside Redis Stream when users are mentioned.

- [ ] **Step 1: Update domain models**
  Modify `SendMessagePayload` in `services/chat-service/internal/domain/models.go` to add `Mentions`:
  ```go
  type SendMessagePayload struct {
      Content   string   `json:"content"`
      ReplyToID string   `json:"reply_to_id"`
      Mentions  []string `json:"mentions,omitempty"`
  }
  ```

- [ ] **Step 2: Add Redis Client to ChatUsecase constructor**
  Modify `services/chat-service/internal/usecase/chat.go`:
  ```go
  import (
      // ...
      "github.com/redis/go-redis/v9"
  )

  type ChatUsecase struct {
      repo *repository.PostgresRepository
      rdb  *redis.Client
  }

  func NewChatUsecase(repo *repository.PostgresRepository, rdb *redis.Client) *ChatUsecase {
      return &ChatUsecase{repo: repo, rdb: rdb}
  }
  ```

- [ ] **Step 3: Update SaveMessage usecase method to publish mentions**
  Modify `SaveMessage` in `services/chat-service/internal/usecase/chat.go` to emit Redis Stream events for mentions:
  ```go
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

      // Publish mentions to Redis stream
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
      return msg, nil
  }
  ```

- [ ] **Step 4: Update usecase invocation in main.go**
  Modify `services/chat-service/cmd/server/main.go` around line 100 to pass `rdb` to usecase constructor:
  ```go
  uc := usecase.NewChatUsecase(repo, rdb)
  ```

- [ ] **Step 5: Verify build**
  Run: `go build ./...` inside `services/chat-service`
  Expected: SUCCESS

- [ ] **Step 6: Commit**
  ```bash
  git add services/chat-service/
  git commit -m "feat(chat): implement mention notifications via Redis stream"
  ```
