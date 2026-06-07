# Playback Service API Specification

Dịch vụ Playback quản lý hàng đợi nhạc (Queue) và đồng bộ hóa trạng thái phát nhạc (Play, Pause, Seek) thời gian thực giữa các thành viên bằng giao thức đồng bộ giống NTP (NTP-like Time Sync Protocol) qua WebSocket.

## Base URL
`/api/v1/rooms/:room_id/playback`

---

## 1. WebSocket Sync Endpoint & Giao thức NTP-like

Để đồng bộ hóa trạng thái âm thanh/video với sai lệch < 50ms, dịch vụ Playback cung cấp kết nối WebSocket để đồng bộ đồng hồ (clock) và trạng thái playback.

*   **URL:** `/ws`
*   **Query Parameters:**
    *   `token`: JWT Access Token hợp lệ.
*   **Kết nối:** `GET /api/v1/rooms/uuid-room-1111/playback/ws?token=access_token`

---

### 1.1 Giao thức Đồng bộ Đồng hồ giống NTP (NTP-like Sync Protocol)

Client và Server sẽ liên tục thực hiện chu kỳ Ping-Pong qua WebSocket (định kỳ 10-15 giây một lần hoặc ngay khi kết nối) để tính toán độ lệch đồng hồ giữa máy Client và máy chủ (Server).

```
  Client                                      Server
    |                                           |
    |--- ping (T1) ---------------------------->| (Nhận tại T2)
    |                                           |
    |<-- pong (T1, T2, T3) ---------------------| (Gửi tại T3)
    |                                           |
 (Nhận tại T4)
```

#### Các mốc thời gian (Timestamp):
*   $T_1$: Thời gian local của Client tại thời điểm gửi tin nhắn `sync:ping`.
*   $T_2$: Thời gian local của Server tại thời điểm nhận được tin nhắn `sync:ping`.
*   $T_3$: Thời gian local của Server tại thời điểm gửi lại tin nhắn `sync:pong`.
*   $T_4$: Thời gian local của Client tại thời điểm nhận được tin nhắn `sync:pong`.

#### Công thức tính toán tại Client:
1.  **Độ trễ mạng khứ hồi (Round Trip Time - RTT):**
    $$RTT = (T_4 - T_1) - (T_3 - T_2)$$
2.  **Độ lệch đồng hồ (Clock Offset - $\theta$):**
    $$\theta = \frac{(T_2 - T_1) + (T_3 - T_4)}{2}$$
3.  **Đồng bộ thời gian thực tế:**
    Thời gian Server chuẩn hóa tại máy Client ($Time_{server\_est}$) sẽ được ước lượng dựa trên thời gian local của Client ($Time_{client\_local}$):
    $$Time_{server\_est} = Time_{client\_local} + \theta$$

#### Cấu trúc Message trao đổi:
*   **Client gửi Ping (`sync:ping`):**
    ```json
    {
      "event": "sync:ping",
      "payload": {
        "t1": 1780824000100 // Local timestamp (ms)
      }
    }
    ```
*   **Server phản hồi Pong (`sync:pong`):**
    ```json
    {
      "event": "sync:pong",
      "payload": {
        "t1": 1780824000100, // Nhận từ client
        "t2": 1780824000120, // Server nhận (ms)
        "t3": 1780824000122  // Server gửi (ms)
      }
    }
    ```

---

### 1.2 Đồng bộ Trạng thái Phát nhạc (Playback State Sync)

Khi một Client gửi lệnh điều khiển (Play/Pause/Seek), Server sẽ xử lý, lưu trạng thái mới vào Redis và broadcast sự kiện `playback:sync` tới toàn bộ client trong phòng.

#### Lệnh điều khiển từ Client -> Server (`playback:control`)
```json
{
  "event": "playback:control",
  "payload": {
    "action": "play", // play, pause, seek
    "track_id": "uuid-track-9999",
    "position_ms": 45000 // Vị trí giây thứ 45
  }
}
```

#### Server Broadcast trạng thái -> Client (`playback:sync`)
```json
{
  "event": "playback:sync",
  "payload": {
    "state": "playing", // playing, paused, stopped
    "current_track_id": "uuid-track-9999",
    "position_ms": 45000,
    "updated_at": 1780824000122 // T3: Thời gian server lúc update state
  }
}
```

#### Công thức tính toán vị trí chơi nhạc hiển thị trên UI Client:
Nếu trạng thái là `"playing"`, client tự động tăng vị trí chạy nhạc theo thời gian thực mà không cần server liên tục gửi event:
$$Position_{current} = position\_ms + (Time_{server\_est} - updated\_at)$$

---

## 2. Các API RESTful cho Hàng đợi (Queue)

### 2.1 Thêm bài hát vào hàng đợi phát
*   **Endpoint:** `POST /queue`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "track_id": "uuid-track-9999"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "queue_id": "uuid-qitem-1111",
        "track_id": "uuid-track-9999",
        "title": "Lofi Chill Beats",
        "added_by": "uuid-user-9999"
      },
      "error": null
    }
    ```

### 2.2 Xóa bài hát khỏi hàng đợi
*   **Endpoint:** `DELETE /queue/:queue_id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã xóa bài hát khỏi hàng đợi."
      },
      "error": null
    }
    ```

### 2.3 Thay đổi vị trí bài hát trong hàng đợi (Kéo thả)
*   **Endpoint:** `PUT /queue/move`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "queue_id": "uuid-qitem-1111",
      "new_position": 2 // Di chuyển lên vị trí thứ 2 trong hàng đợi
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã sắp xếp lại hàng đợi thành công."
      },
      "error": null
    }
    ```
