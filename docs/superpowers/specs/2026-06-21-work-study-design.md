# Specification: Phase 3 — Work & Study Features

**Tạo:** 2026-06-21  
**Trạng thái:** Chờ duyệt  
**Tác giả:** Antigravity  

---

## 1. Mục tiêu & Phạm vi
Phase 3 tập trung vào tính năng hỗ trợ học tập và làm việc nhóm hiệu quả trong phòng:
1. **Focus Timer (`timer-service`)**: Đồng hồ Pomodoro đồng bộ toàn phòng, điều khiển phát/dừng nhạc.
2. **Breakout Rooms (`room-service`)**: Tạo phòng thảo luận nhóm nhỏ, cô lập voice/video và chat.
3. **Collaborative Notes (`collab-service`)**: Soạn thảo ghi chú & checklist thời gian thực.
4. **Co-queue Round-Robin Fairness (`playback-service`)**: Thuật toán chọn bài hát tiếp theo xoay vòng và theo phiếu bầu.

---

## 2. Kiến trúc & Phân vùng Domain (Bounded Context)

```
                       ┌─────────────────────────┐
                       │       API Gateway       │
                       │         (8080)          │
                       └───────────┬─────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │ (HTTP / WS)             │ (gRPC)                  │ (HTTP / WS)
   ┌─────▼──────┐            ┌─────▼──────┐            ┌─────▼──────┐
   │room-service│            │playback-svc│            │collab-serv │
   │   (8083)   │            │   (8087)   │            │   (8094)   │
   └─────┬──────┘            └─────▲──────┘            └─────┬──────┘
         │                         │                         │
         │                         │ (gRPC)                  │
         │                   ┌─────┴──────┐                  │
         │                   │ timer-serv │                  │
         │                   │   (8093)   │                  │
         │                   └─────┬──────┘                  │
   ┌─────▼──────┐            ┌─────▼──────┐            ┌─────▼──────┐
   │ PostgreSQL │            │   Redis    │            │ PostgreSQL │
   │ (syncspace)│            │  (Cache)   │            │  (collab)  │
   └────────────┘            └────────────┘            └────────────┘
```

### Dịch vụ mới & Port cấu hình
1. **`timer-service`**: Port HTTP `8093`, gRPC `9093`.
2. **`collab-service`**: Port HTTP `8094`.

---

## 3. Chi tiết Thiết kế Kỹ thuật

### 3.1. Focus Timer (`timer-service`)
* **Lưu trữ Redis**:
  - Trạng thái timer: `room:<room_id>:pomodoro` (JSON string)
    ```json
    {
      "status": "focus" | "break" | "paused" | "idle",
      "duration_seconds": 1500,
      "remaining_seconds": 1500,
      "ends_at": 1782012345000,
      "current_cycle": 1,
      "total_cycles": 4
    }
    ```
* **Lệnh WebSocket**:
  - `timer:start`: `{ "duration_seconds": 1500, "cycles": 4 }`
  - `timer:pause`
  - `timer:resume`
  - `timer:stop`
* **gRPC Integration**:
  - Protobuf định nghĩa dịch vụ internal của `playback-service` (`services/playback-service/api/v1/playback.proto`):
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

---

### 3.2. Breakout Rooms (`room-service`)
* **Database Schema Migration (`000003_add_subrooms.up.sql`)**:
  ```sql
  ALTER TABLE rooms ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36) REFERENCES rooms(id) ON DELETE CASCADE;
  ALTER TABLE room_members ADD COLUMN IF NOT EXISTS active_sub_room_id VARCHAR(36) REFERENCES rooms(id) ON DELETE SET NULL;
  ```
* **REST APIs**:
  - `POST /api/v1/rooms/:id/subrooms`: Tạo sub-room.
  - `GET /api/v1/rooms/:id/subrooms`: Lấy danh sách sub-room.
  - `PUT /api/v1/rooms/:id/members/move`: Di chuyển thành viên vào/ra sub-room.
* **Quy trình hoạt động Voice & Chat**:
  1. Khi được chuyển vào sub-room, client nhận thông báo từ WebSocket.
  2. Client ngắt LiveKit room hiện tại.
  3. Client gọi `voice-service` lấy token mới với tên room con: `room-<parent_id>-sub-<sub_id>`.
  4. Client mute âm lượng nhạc của player chính.

---

### 3.3. Collaborative Notes (`collab-service`)
* **Database Schema (`000001_init_collab.up.sql`)**:
  ```sql
  CREATE TABLE IF NOT EXISTS notes (
      id VARCHAR(36) PRIMARY KEY,
      room_id VARCHAR(36) NOT NULL,
      title VARCHAR(100) NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
  );
  
  CREATE TABLE IF NOT EXISTS note_blocks (
      id VARCHAR(36) PRIMARY KEY,
      note_id VARCHAR(36) REFERENCES notes(id) ON DELETE CASCADE,
      block_type VARCHAR(10) NOT NULL,
      content TEXT NOT NULL,
      is_checked BOOLEAN DEFAULT FALSE,
      order_index INT NOT NULL,
      updated_by VARCHAR(36) NOT NULL,
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
  );
  ```
* **WebSocket Events**:
  - Client gửi: `block:add`, `block:update`, `block:delete`, `block:move`.
  - Server gửi broadcast: `block:created`, `block:updated`, `block:deleted`, `block:ordered`.

---

### 3.4. Co-queue Round-Robin + Votes Algorithm (`playback-service`)
* **Thuật toán sắp xếp lượt phát tiếp theo (`advanceToNextTrack`)**:
  1. Nếu có Poll Winner hợp lệ -> Chọn phát Poll Winner.
  2. Nếu không có Poll:
     - Gom danh sách track còn lại theo `added_by`.
     - Sắp xếp track của từng user theo: `votes DESC`, `position ASC`.
     - Xác định lượt của user tiếp theo (user có bài hát thêm lâu nhất sẽ được xếp lượt trước).
     - Chọn bài hát đầu tiên trong tập của user tới lượt.
     - Di chuyển bài hát được chọn lên đầu danh sách phát (gọi API `playlist-service` di chuyển vị trí).

---

## 4. Kế hoạch xác thực (Verification Plan)

### Kiểm thử tự động
- Viết unit test cho thuật toán Round-Robin trong `playback-service`.
- Viết unit test cho parser/event handler của `collab-service`.
- Chạy toàn bộ test suites frontend và backend.

### Kiểm thử thủ công
- Tạo kịch bản chia phòng và họp voice/video nhóm con.
- Kiểm tra tính đồng bộ đếm ngược Pomodoro và pause nhạc khi chuyển Break.
