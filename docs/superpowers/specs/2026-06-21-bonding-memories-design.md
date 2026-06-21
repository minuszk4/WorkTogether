# Specification: Phase 4 — Bonding & Memories (Gắn Kết & Ký Ức)

**Tạo:** 2026-06-21  
**Trạng thái:** Chờ duyệt  
**Tác giả:** Antigravity  

---

## 1. Mục tiêu & Phạm vi
Giai đoạn này tập trung vào việc tạo bản sắc cho phòng làm việc và lưu giữ những kỷ niệm hoạt động chung giữa các thành viên:
1. **Room Identity (Bản sắc phòng)**: Cho phép chủ phòng/moderator cập nhật ảnh đại diện (avatar) của phòng, thiết lập các quy định (rules), và lựa chọn giao diện màu sắc/tâm trạng (theme/mood).
2. **Shared Listening History (Lịch sử nghe chung)**: Tự động ghi lại lịch sử các bài nhạc đã được phát hoàn thành hoặc bắt đầu phát trong phòng.
3. **Room Stats (Thống kê hoạt động)**: Cung cấp số liệu tổng quan về thời gian nghe nhạc chung, số lượng bài hát đã phát và những bài hát được ưa thích nhất (phát nhiều nhất).

---

## 2. Kiến trúc & Sơ đồ luồng dữ liệu (Data Flow)

### 2.1. Room Identity
* Lưu trữ trực tiếp trong cơ sở dữ liệu `worktogether_room` (bảng `rooms`).
* Endpoint `PUT /api/v1/rooms/:id/settings` được mở rộng để cập nhật các thuộc tính mới.

### 2.2. Shared Listening History & Stats
* Khi một bài hát mới bắt đầu phát trong `playback-service` (hoặc chuyển bài ở `advanceToNextTrack`), `playback-service` sẽ gọi HTTP POST sang `music-service` tại endpoint `/api/v1/music/history`.
* `music-service` lưu trữ lịch sử vào bảng `history` trong database `worktogether_music`.
* API lấy lịch sử (`GET /api/v1/music/history/:room_id`) và thống kê (`GET /api/v1/music/stats/:room_id`) sẽ được phục vụ bởi `music-service`.

```
┌─────────────────┐             HTTP POST             ┌───────────────┐
│                 ├──────────────────────────────────►│               │
│playback-service │  /api/v1/music/history           │ music-service │
│                 │  {room_id, track_id}              │    (8085)     │
└────────┬────────┘                                   └───────┬───────┘
         │                                                    │
         ▼ (Redis State)                                      ▼ (Postgres DB)
    [Playback Sync]                                    [worktogether_music]
```

---

## 3. Chi tiết Thiết kế Kỹ thuật

### 3.1. Thay đổi Cơ sở dữ liệu (Database Schema Migrations)

#### `room-service` (`worktogether_room` DB)
Tạo file migration `000004_add_room_identity.up.sql`:
```sql
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS rules TEXT;
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS theme VARCHAR(50) DEFAULT 'cool-ocean';
```

#### `music-service` (`worktogether_music` DB)
Bảng `history` đã tồn tại sẵn trong code khởi tạo tự động của `music-service`. Chúng ta sẽ thêm các câu lệnh SQL để truy vấn thống kê (Stats):
* **Tổng số bài đã phát**: `SELECT COUNT(*) FROM history WHERE room_id = $1`
* **Tổng thời lượng đã phát (ms)**: `SELECT COALESCE(SUM(t.duration_ms), 0) FROM history h JOIN tracks t ON h.track_id = t.id WHERE h.room_id = $1`
* **Top bài hát phát nhiều nhất**:
  ```sql
  SELECT t.id, t.title, t.artist, t.thumbnail_url, COUNT(h.id) AS play_count
  FROM history h
  JOIN tracks t ON h.track_id = t.id
  WHERE h.room_id = $1
  GROUP BY t.id, t.title, t.artist, t.thumbnail_url
  ORDER BY play_count DESC
  LIMIT $2
  ```

---

### 3.2. REST APIs & Interfaces

#### 1. `room-service`
* **`PUT /api/v1/rooms/:id/settings`** (Cập nhật Identity)
  * Payload mở rộng:
    ```json
    {
      "name": "Tên phòng mới",
      "description": "Mô tả phòng",
      "add_music_policy": "all",
      "avatar_url": "https://example.com/avatar.png",
      "rules": "Không spam nhạc rác, tập trung học tập.",
      "theme": "cool-ocean"
    }
    ```

#### 2. `music-service`
* **`GET /api/v1/music/history/:room_id`**: Lấy danh sách lịch sử phát nhạc gần đây (mặc định 20 bài).
* **`GET /api/v1/music/rooms/:room_id/stats`**: Lấy thống kê của phòng.
  * Phản hồi mẫu:
    ```json
    {
      "success": true,
      "data": {
        "total_tracks_played": 154,
        "total_play_time_ms": 32840000,
        "top_tracks": [
          {
            "id": "track-uuid-1",
            "title": "Lofi Chill Music",
            "artist": "Lofi Boy",
            "thumbnail_url": "https://example.com/thumb.jpg",
            "play_count": 14
          }
        ]
      }
    }
    ```

---

### 3.3. Web Client Components & UI Integration

Chúng ta sẽ mở rộng giao diện của phòng bằng cách bổ sung một tab **"Bản sắc & Kỷ ức" (Identity & Stats)** bên trong Panel hoặc Drawer bên cạnh Chat/Notes.

1. **Room Identity Editor Modal**:
   * Chỉ hiển thị cho Host (Owner) hoặc Moderator.
   * Cho phép chỉnh sửa URL ảnh đại diện, viết nội quy phòng (Rules), chọn Theme (Cool Ocean, Sunset Glow, Emerald Forest, Midnight Dark).
2. **Room Details Panel**:
   * Hiển thị Avatar phòng lớn ở vị trí Header.
   * Hiển thị bảng nội quy phòng (Rules) để mọi người có thể xem nhanh.
3. **Listening History & Stats Component**:
   * Hiển thị các chỉ số thống kê trực quan (tổng thời gian nghe chung định dạng hh:mm:ss, tổng bài hát).
   * Danh sách Top 5 bài hát được nghe nhiều nhất có nút Quick-add để đưa nhanh bài hát đó trở lại hàng đợi.
   * Danh sách 20 bài hát đã phát gần đây dạng timeline với thời gian phát nhạc cụ thể.

---

## 4. Kế hoạch xác thực (Verification Plan)

### Kiểm thử tự động
- Viết unit test cho các truy vấn SQL thống kê (Stats) trong `music-service/internal/repository/postgres_test.go`.
- Viết unit test cho hàm lưu lịch sử phát nhạc của `playback-service`.

### Kiểm thử thủ công
- Mở rộng cài đặt phòng, lưu avatar, rules và theme. Kiểm tra xem giao diện đổi màu tương ứng khi thay đổi theme hay không.
- Phát nhạc liên tục qua một vài bài hát để kiểm tra xem lịch sử phát nhạc có được ghi nhận đúng vào database `worktogether_music` hay không.
- Mở tab thống kê để xác nhận các chỉ số tổng thời gian phát và top bài hát hiển thị chính xác.
