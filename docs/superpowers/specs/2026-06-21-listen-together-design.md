# Phase 2: Nghe Cùng Nhau — Technical Design Specification

**Tạo:** 2026-06-21  
**Trạng thái:** Draft  
**Tác giả:** Antigravity  

---

## 1. Goal

Phase 2 nhằm tăng cường sự tương tác và gắn kết giữa các thành viên trong phòng thông qua trải nghiệm âm nhạc đồng bộ chiều sâu:
1. **Synced Lyrics**: Lời bài hát trượt động đồng bộ theo thời gian thực với nhạc đang phát.
2. **Bookmark moments**: Cho phép lưu các khoảnh khắc (timestamp + ghi chú) và chia sẻ nhanh vào Chat dưới dạng liên kết nhảy trực tiếp đến giây đó.
3. **Up-next realtime poll**: Tự động bình chọn bài hát tiếp theo trong 30 giây cuối cùng của bài hát hiện tại.
4. **DJ mode / takeover**: Host có thể tạm thời nhường quyền điều khiển nhạc (Play/Pause/Seek) cho một Guest DJ khác.

---

## 2. Database Schema

Để bảo vệ tính cô lập dữ liệu (database isolation) của các microservices, toàn bộ bảng liên quan đến bài hát sẽ được đặt trong PostgreSQL của `music-service` (do các bảng này có ràng buộc khóa ngoại tới bảng `tracks` vốn thuộc `music-service`).

### 2.1 Bảng Lời bài hát (`track_lyrics`)
Lưu trữ lời bài hát dưới định dạng LRC chuẩn có chứa timestamp của từng dòng.
```sql
CREATE TABLE IF NOT EXISTS track_lyrics (
    track_id VARCHAR(36) PRIMARY KEY,
    content TEXT NOT NULL, -- Nội dung định dạng LRC chuẩn: [00:10.50]Dòng lời đầu tiên...
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_track FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
);
```

### 2.2 Bảng Bookmark khoảnh khắc (`bookmarks`)
Lưu trữ các bookmark cá nhân hoặc tập thể trong một phòng cụ thể.
```sql
CREATE TABLE IF NOT EXISTS bookmarks (
    id VARCHAR(36) PRIMARY KEY,
    room_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    track_id VARCHAR(36) NOT NULL,
    position_ms INTEGER NOT NULL,
    note VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_track_bookmark FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_bookmarks_room ON bookmarks(room_id);
```

---

## 3. API Endpoints (`music-service`)

### 3.1 REST API cho Lyrics

#### Lấy lời bài hát
- **Endpoint**: `GET /api/v1/music/tracks/:track_id/lyrics`
- **Response (200 OK - Có lời)**:
  ```json
  {
    "success": true,
    "data": {
      "track_id": "uuid-track-123",
      "content": "[00:00.00]Intro\n[00:10.00]Dòng thứ nhất\n[00:15.50]Dòng thứ hai",
      "updated_at": "2026-06-21T02:00:00Z"
    },
    "error": null
  }
  ```
- **Response (404 Not Found - Chưa có lời)**:
  ```json
  {
    "success": false,
    "data": null,
    "error": {
      "code": "LYRICS_NOT_FOUND",
      "message": "Bài hát này chưa có lời."
    }
  }
  ```

#### Thêm / Chỉnh sửa lời bài hát
- **Endpoint**: `POST /api/v1/music/tracks/:track_id/lyrics`
- **Request Body**:
  ```json
  {
    "content": "[00:00.00]Intro\n[00:10.00]Dòng thứ nhất\n[00:15.50]Dòng thứ hai"
  }
  ```
- **Response (200 OK)**:
  ```json
  {
    "success": true,
    "data": {
      "track_id": "uuid-track-123",
      "message": "Cập nhật lời bài hát thành công."
    },
    "error": null
  }
  ```

---

### 3.2 REST API cho Bookmarks

#### Tạo bookmark mới
- **Endpoint**: `POST /api/v1/music/rooms/:room_id/bookmarks`
- **Headers**: `Authorization: Bearer <token>`
- **Request Body**:
  ```json
  {
    "track_id": "uuid-track-123",
    "position_ms": 75000,
    "note": "đoạn solo guitar đỉnh cực"
  }
  ```
- **Response (201 Created)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "uuid-bookmark-456",
      "room_id": "uuid-room-789",
      "user_id": "uuid-user-999",
      "track_id": "uuid-track-123",
      "position_ms": 75000,
      "note": "đoạn solo guitar đỉnh cực",
      "created_at": "2026-06-21T02:05:00Z"
    },
    "error": null
  }
  ```

#### Lấy danh sách bookmark trong phòng
- **Endpoint**: `GET /api/v1/music/rooms/:room_id/bookmarks`
- **Headers**: `Authorization: Bearer <token>`
- **Response (200 OK)**:
  ```json
  {
    "success": true,
    "data": [
      {
        "id": "uuid-bookmark-456",
        "user_id": "uuid-user-999",
        "track_id": "uuid-track-123",
        "position_ms": 75000,
        "note": "đoạn solo guitar đỉnh cực",
        "created_at": "2026-06-21T02:05:00Z",
        "track": {
          "title": "Hotel California",
          "artist": "Eagles",
          "thumbnail_url": "https://..."
        }
      }
    ],
    "error": null
  }
  ```

#### Xóa bookmark
- **Endpoint**: `DELETE /api/v1/music/rooms/:room_id/bookmarks/:id`
- **Headers**: `Authorization: Bearer <token>`
- **Response (200 OK)**:
  ```json
  {
    "success": true,
    "data": {
      "message": "Đã xóa bookmark thành công."
    },
    "error": null
  }
  ```

---

## 4. WebSocket Events (`playback-service` + Redis)

### 4.1 DJ Mode / Takeover Protocol

Quyền điều khiển trình phát được lưu trữ trong Redis với khóa `room:<room_id>:guest_dj` có TTL (mặc định 180 giây).

#### Cấp quyền Guest DJ (Host -> Server REST API)
- **Endpoint**: `POST /api/v1/rooms/:room_id/dj`
- **Body**: `{ "user_id": "uuid-guest-111", "duration_seconds": 180 }`

#### Thu hồi quyền Guest DJ (Host -> Server REST API)
- **Endpoint**: `DELETE /api/v1/rooms/:room_id/dj`

#### Server Broadcast khi có DJ mới (`dj:takeover`)
```json
{
  "event": "dj:takeover",
  "payload": {
    "user_id": "uuid-guest-111",
    "display_name": "Nguyen Van B",
    "ends_at": 1780824000500 -- Unix timestamp (ms) khi hết quyền
  }
}
```

#### Server Broadcast khi giải phóng quyền DJ (`dj:released`)
```json
{
  "event": "dj:released",
  "payload": {
    "reason": "expired" -- "expired" hoặc "revoked"
  }
}
```

> [!IMPORTANT]
> Khi Redis có key `room:<room_id>:guest_dj`, bất cứ sự kiện điều khiển `playback:control` nào gửi lên từ Client sẽ bị Server kiểm tra. Nếu `user_id` gửi lệnh không trùng với `guest_dj` và cũng không phải là Room Host (Owner), Server sẽ từ chối và phản hồi lại thông báo lỗi.

---

### 4.2 Up-Next Realtime Poll Protocol

Khi bài hát hiện tại có độ dài $D$ (ms) và thời gian phát đạt mốc $D - 30000$ (còn lại 30s):

#### Server kích hoạt Poll (`poll:start`)
Server chọn 3 bài hát xếp đầu trong hàng đợi và broadcast:
```json
{
  "event": "poll:start",
  "payload": {
    "poll_id": "uuid-poll-999",
    "options": [
      { "track_id": "track-aaa", "title": "Song A", "artist": "Artist A" },
      { "track_id": "track-bbb", "title": "Song B", "artist": "Artist B" },
      { "track_id": "track-ccc", "title": "Song C", "artist": "Artist C" }
    ],
    "ends_at": 1780824000300 -- Unix timestamp (ms) kết thúc bài hiện tại
  }
}
```

#### Client gửi Vote (`poll:vote` - Client -> Server WS)
```json
{
  "event": "poll:vote",
  "payload": {
    "track_id": "track-aaa"
  }
}
```

#### Server cập nhật số lượng Vote (`poll:update`)
```json
{
  "event": "poll:update",
  "payload": {
    "votes": {
      "track-aaa": 5,
      "track-bbb": 2,
      "track-ccc": 0
    }
  }
}
```

#### Kết thúc Poll và đổi bài
Khi bài hát kết thúc:
1. Server chọn `track_id` có số phiếu cao nhất.
2. Sắp xếp lại hàng đợi: Đẩy bài thắng cuộc lên đầu hàng đợi phát.
3. Broadcast `poll:end` tới toàn phòng và kích hoạt phát bài hát đó qua sự kiện `playback:sync` bình thường.

---

## 5. UI/UX Flows

### 5.1 Synced Lyrics Flow
1. Khi bài hát bắt đầu phát, Client gửi yêu cầu `GET /lyrics` của `track_id` đó.
2. Nếu có, Client phân tích nội dung LRC thành mảng cấu trúc:
   ```typescript
   interface LyricLine {
     timeMs: number;
     text: string;
   }
   ```
3. Trong vòng lặp render, so sánh `position_ms` của trình phát với danh sách lyric. Dòng lyric đang phát sẽ có `timeMs` lớn nhất nhưng nhỏ hơn hoặc bằng `position_ms`.
4. Dòng lyric được highlight bằng hiệu ứng đổi màu (Cool Ocean Cyan) và kích thước chữ lớn hơn. Phần hiển thị lyric sẽ tự động trượt mượt mà bằng CSS `scroll-into-view`.
5. Nếu bài hát chưa có lời, hiển thị một nút trống "Thêm lời bài hát". Click vào mở modal cho phép người dùng dán lời định dạng LRC.

### 5.2 Click-to-Seek từ Chat Flow
1. Người dùng nhấn nút share trên một Bookmark ở danh sách Sidebar.
2. Client gửi tin nhắn Chat qua WebSocket có cấu trúc chứa thông tin đặc biệt:
   - Text: `[02:15] - phần dạo nhạc đỉnh cao`
   - Metadata: `{ "type": "seek_link", "position_ms": 135000 }`
3. Khi Chat Component hiển thị tin nhắn, nếu tin nhắn có metadata `seek_link`, nó sẽ render đoạn mốc thời gian thành một liên kết dạng nút bấm (ví dụ màu Ocean Teal).
4. Khi người dùng bất kỳ click vào nút bấm đó:
   - Kiểm tra quyền: Nếu họ là Host hoặc Guest DJ, client sẽ gửi lệnh WS `playback:control` với action `seek` và `position_ms: 135000` để đồng bộ tua nhạc cho toàn bộ phòng.
   - Nếu không có quyền DJ, trình phát nhạc của riêng cá nhân họ sẽ thực hiện tua nhạc (Local Seek).

---

## 6. Verification Plan

### 6.1 Automated Tests
- Bổ sung unit tests trong Go cho các handler REST API (Lyrics & Bookmarks) trong `music-service`.
- Bổ sung unit tests cho cơ chế quản lý Guest DJ và Realtime Polls trong `playback-service`.
- Chạy toàn bộ test suite frontend và backend:
  ```bash
  # Web Client Tests
  npm run test -- --watch=false --browsers=ChromeHeadless
  
  # Go Microservice Tests
  go test ./...
  ```

### 6.2 Manual Verification
1. Mở phòng, thêm bài hát, dán lời bài hát LRC và kiểm tra xem lời trượt chuẩn khớp với nhịp phát nhạc.
2. Tạo 1 bookmark ở giây thứ 50, nhấn nút chia sẻ vào chat, và kiểm tra xem tin nhắn hiển thị link bấm seek nhạc chuẩn xác hay không.
3. Thêm 3 bài hát vào hàng đợi. Chờ đến 30 giây cuối cùng, kiểm tra xem Poll bình chọn có tự động nhảy lên không. Thực hiện bầu chọn bằng 2 tài khoản khác nhau để xác nhận bài có số phiếu cao nhất sẽ được phát tiếp theo.
4. Host phân quyền Guest DJ cho thành viên B, kiểm tra xem thành B có nút bấm điều khiển nhạc còn thành viên C thì bị khoá hoàn toàn.
