# Phase 3: Work & Study Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Triển khai các tính năng Focus Timer Pomodoro, Breakout Rooms (Sub Rooms), Collaborative Notes và thuật toán Co-queue Round-Robin công bằng cho cả Backend (Go) và Frontend (Angular).

**Architecture:** Bổ sung `timer-service` (Go + Redis) và `collab-service` (Go + Postgres), nâng cấp `room-service` (Sub-rooms model & APIs) và `playback-service` (gRPC Endpoint & Round-robin Selection). Cập nhật Angular Client UI tương ứng.

**Tech Stack:** Go (Gin, sqlx, go-redis, grpc), Angular 19, LiveKit (voice/video sub-rooms).

## Global Constraints
- Viết mã nguồn sạch, tuân thủ kiến trúc phân lớp sạch (Clean Architecture) cho các dịch vụ Go.
- Giao diện người dùng tuân thủ HSL Cool Ocean Design System, kế thừa styles từ phòng chính.
- Mọi API, schema database mới phải đi kèm migration và unit test đầy đủ.

---

### Task 1: Room-service Sub-rooms Migration & REST API

**Files:**
- Create: `services/room-service/db/migrations/000003_add_subrooms.up.sql`
- Modify: `services/room-service/internal/domain/models.go`
- Modify: `services/room-service/internal/repository/postgres.go`
- Modify: `services/room-service/internal/usecase/room.go`
- Modify: `services/room-service/internal/delivery/http/handlers.go`

**Interfaces:**
- Consumes: Postgres connection.
- Produces: REST APIs:
  - `POST /api/v1/rooms/:id/subrooms` (tạo sub-room)
  - `GET /api/v1/rooms/:id/subrooms` (liệt kê sub-rooms)
  - `PUT /api/v1/rooms/:id/members/move` (di chuyển thành viên)

- [ ] **Step 1: Write DB Migration SQL**
  Tạo tệp `services/room-service/db/migrations/000003_add_subrooms.up.sql` với:
  ```sql
  ALTER TABLE rooms ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36) REFERENCES rooms(id) ON DELETE CASCADE;
  ALTER TABLE room_members ADD COLUMN IF NOT EXISTS active_sub_room_id VARCHAR(36) REFERENCES rooms(id) ON DELETE SET NULL;
  ```

- [ ] **Step 2: Update models.go**
  Cập nhật struct `Room` và `RoomMember` trong `services/room-service/internal/domain/models.go` để hỗ trợ các trường mới:
  ```go
  // Thêm vào struct Room:
  ParentID *string `json:"parent_id"`

  // Thêm vào struct RoomMember:
  ActiveSubRoomID *string `json:"active_sub_room_id"`
  ```

- [ ] **Step 3: Update postgres.go for sub-rooms queries**
  Thêm logic truy vấn vào `services/room-service/internal/repository/postgres.go`:
  ```go
  func (r *PostgresRepository) CreateSubRoom(ctx context.Context, sub *domain.Room) error {
      query := `INSERT INTO rooms (id, name, description, privacy, owner_id, parent_id, created_at, updated_at) 
                VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`
      _, err := r.db.ExecContext(ctx, query, sub.ID, sub.Name, sub.Description, sub.Privacy, sub.OwnerID, sub.ParentID)
      return err
  }

  func (r *PostgresRepository) GetSubRooms(ctx context.Context, parentID string) ([]*domain.Room, error) {
      query := `SELECT id, name, description, privacy, owner_id, parent_id, created_at, updated_at FROM rooms WHERE parent_id = $1`
      rows, err := r.db.QueryContext(ctx, query, parentID)
      if err != nil { return nil, err }
      defer rows.Close()
      var list []*domain.Room
      for rows.Next() {
          var rm domain.Room
          if err := rows.Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.OwnerID, &rm.ParentID, &rm.CreatedAt, &rm.UpdatedAt); err != nil {
              return nil, err
          }
          list = append(list, &rm)
      }
      return list, nil
  }

  func (r *PostgresRepository) MoveMember(ctx context.Context, roomID string, userID string, subRoomID *string) error {
      query := `UPDATE room_members SET active_sub_room_id = $1 WHERE room_id = $2 AND user_id = $3`
      _, err := r.db.ExecContext(ctx, query, subRoomID, roomID, userID)
      return err
  }
  ```

- [ ] **Step 4: Update Usecase & Handlers**
  Thêm hàm Usecase tương ứng và đăng ký routes trong `services/room-service/internal/delivery/http/handlers.go`.
  Chạy lệnh để restart room-service và xác minh API.

---

### Task 2: Playback-service gRPC Server & Round-Robin Queue Selection

**Files:**
- Create: `services/playback-service/api/v1/playback.proto`
- Modify: `services/playback-service/internal/delivery/http/ws_hub.go`
- Modify: `services/playback-service/internal/usecase/playback.go`

**Interfaces:**
- Consumes: Playlist tracks from `playlist-service`.
- Produces: gRPC service `PlaybackInternalService` and updated `advanceToNextTrack` algorithm.

- [ ] **Step 1: Write Protobuf specification**
  Tạo tệp `services/playback-service/api/v1/playback.proto` chứa:
  ```protobuf
  syntax = "proto3";
  package worktogether.playback.v1;
  option go_package = "github.com/worktogether/services/playback-service/api/v1;v1";

  service PlaybackInternalService {
    rpc SetPlaybackStateByTimer(SetPlaybackStateRequest) returns (SetPlaybackStateResponse);
  }

  message SetPlaybackStateRequest {
    string room_id = 1;
    string action = 2; // "pause", "resume"
  }

  message SetPlaybackStateResponse {
    bool success = 1;
  }
  ```

- [ ] **Step 2: Generate gRPC code**
  Chạy lệnh để sinh file Go từ proto file:
  `protoc --go_out=. --go-grpc_out=. services/playback-service/api/v1/playback.proto`

- [ ] **Step 3: Implement Round-Robin logic in ws_hub.go**
  Thay đổi thuật toán chọn bài tiếp theo tại `advanceToNextTrack` trong `ws_hub.go`:
  Nhóm bài hát theo `added_by`, sắp xếp bài hát từng user theo `votes DESC`, xoay vòng lượt phát theo thứ tự thêm đầu tiên của user đó.

---

### Task 3: Timer-service Scaffolding, WebSocket & Playback Integration

**Files:**
- Create: `services/timer-service/cmd/server/main.go`
- Create: `services/timer-service/internal/delivery/ws/handler.go`
- Create: `services/timer-service/internal/usecase/timer.go`

**Interfaces:**
- Consumes: Redis and gRPC call to `playback-service`.
- Produces: WebSocket server on port `8093` for Pomodoro.

- [ ] **Step 1: Scaffolding main.go for timer-service**
  Tạo tệp khởi chạy Gin HTTP server và kết nối Redis Client.
  
- [ ] **Step 2: Implement WebSocket Timer State Broadcast**
  Bản tin đếm ngược theo giây được gửi từ `timer-service` qua kênh WS cho cả phòng.

---

### Task 4: Collab-service Scaffolding, DB Schema & WebSocket Notes Sync

**Files:**
- Create: `services/collab-service/db/migrations/000001_init_collab.up.sql`
- Create: `services/collab-service/cmd/server/main.go`
- Create: `services/collab-service/internal/delivery/ws/handler.go`

**Interfaces:**
- Consumes: Postgres for note blocks.
- Produces: Realtime document blocks sync API.

- [ ] **Step 1: Write collab migration schema**
  Tạo bảng `notes` và `note_blocks` lưu trữ nội dung ghi chú / checklist theo hàng rời rạc.

---

### Task 5: Web Client Breakout Rooms UI & Voice Switching

**Files:**
- Create: `web-client/src/app/core/services/subroom.service.ts`
- Create: `web-client/src/app/features/room/components/subrooms/subrooms.component.ts`
- Create: `web-client/src/app/features/room/components/subrooms/subrooms.component.html`

**Interfaces:**
- Consumes: Room service HTTP REST APIs.
- Produces: Panel control to create/delete sub-rooms and auto LiveKit voice reconnecting.

---

### Task 6: Web Client Focus Timer UI & Controls

**Files:**
- Create: `web-client/src/app/features/room/components/timer/timer.component.ts`
- Create: `web-client/src/app/features/room/components/timer/timer.component.html`

**Interfaces:**
- Consumes: Timer service WebSocket connection.
- Produces: Pomodoro overlay UI with start/pause/stop buttons.

---

### Task 7: Web Client Collaborative Notes & Checklist Component

**Files:**
- Create: `web-client/src/app/features/room/components/collab-notes/collab-notes.component.ts`
- Create: `web-client/src/app/features/room/components/collab-notes/collab-notes.component.html`

**Interfaces:**
- Consumes: Collab service WebSocket events.
- Produces: Realtime rich checklist board inside room layout.
