# Voice Service API Specification

Dịch vụ Voice chịu trách nhiệm tích hợp hệ thống gọi thoại (Voice Call) thời gian thực bằng LiveKit SFU Server. Dịch vụ này thực hiện cấp phát Access Token bảo mật cho Client và lắng nghe sự kiện từ LiveKit Server qua Webhooks.

## Base URL
`/api/v1/voice`

---

## 1. Lấy Token kết nối Voice Channel
Client gọi API này để nhận thông tin định tuyến và Access Token đã được ký số bằng LiveKit API Key. Dùng Token này để thiết lập kết nối WebRTC trực tiếp tới LiveKit Server.

*   **Endpoint:** `GET /rooms/:room_id/token`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "livekit_url": "wss://sfu.worktogether.com",
        "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2aWRlbyI6e30sIm11dGVkIjpmYWxzZX0..."
      },
      "error": null
    }
    ```
*   **Quy trình xử lý nội bộ:**
    1.  `voice-service` nhận request, gọi sang `room-service` (via gRPC) để kiểm tra xem `user_id` có phải thành viên của `room_id` và có quyền sử dụng thoại không (`can_use_voice` flag).
    2.  Nếu hợp lệ, sử dụng LiveKit SDK để khởi tạo Access Grant với các quyền:
        *   `RoomJoin`: true
        *   `Room`: `room_id`
        *   `Identity`: `user_id`
        *   `CanPublish`: true
        *   `CanSubscribe`: true
    3.  Trả về JWT Token được ký bằng `LiveKit Secret Key`.

---

## 2. LiveKit Webhook Receiver
Endpoint nhận sự kiện (Webhook) từ LiveKit Server để cập nhật trạng thái kết nối thoại vào hệ thống cơ sở dữ liệu và broadcast trạng thái tới các service khác qua Redis Streams.

*   **Endpoint:** `POST /webhooks`
*   **Headers:**
    *   `Authorization: <LiveKit-Signed-Header>` (Dùng để xác thực tính hợp lệ của Webhook từ LiveKit).
*   **Request Body (Mẫu Event Participant Joined):**
    ```json
    {
      "event": "participant_joined",
      "room": {
        "name": "uuid-room-1111",
        "sid": "RM_xxxxxx"
      },
      "participant": {
        "identity": "uuid-user-9999",
        "state": "ACTIVE",
        "joined_at": 1780824000
      }
    }
    ```
*   **Request Body (Mẫu Event Participant Left):**
    ```json
    {
      "event": "participant_left",
      "room": {
        "name": "uuid-room-1111",
        "sid": "RM_xxxxxx"
      },
      "participant": {
        "identity": "uuid-user-9999"
      }
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "message": "Webhook processed successfully"
    }
    ```
*   **Quy trình xử lý sự kiện Webhook:**
    *   Khi có sự kiện `participant_joined` hoặc `participant_left`, `voice-service` sẽ phát một Event tương ứng vào Redis Streams (`stream:voice_events`) để các client nhận biết ai đang bật/tắt thoại trong phòng thông qua websocket.
