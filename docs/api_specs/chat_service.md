# Chat Service API Specification

Dịch vụ Chat xử lý tin nhắn văn bản tức thời, tương tác cảm xúc tin nhắn (reactions), ghim tin nhắn (pins) qua giao tiếp REST và WebSocket.

## Base URL
`/api/v1/rooms/:room_id/chat`

---

## 1. WebSocket Endpoint
Tạo kết nối thời gian thực phục vụ chat và sự kiện tương tác.

*   **URL:** `/ws`
*   **Query Parameters:**
    *   `token`: JWT Access Token hợp lệ.
*   **Giao thức kết nối:**
    *   Client gửi HTTP request nâng cấp (Upgrade) lên WebSocket:
        `GET /api/v1/rooms/uuid-room-1111/chat/ws?token=access_token`

### Các Sự kiện WebSocket Client Gửi (Client -> Server)

#### 1. Gửi tin nhắn (`chat:send_message`)
```json
{
  "event": "chat:send_message",
  "payload": {
    "content": "Hello world! @john_doe",
    "reply_to_id": "optional-uuid-reply-msg"
  }
}
```

#### 2. Đang soạn tin nhắn (`chat:typing`)
```json
{
  "event": "chat:typing",
  "payload": {
    "is_typing": true
  }
}
```

#### 3. Tương tác Emoji (`chat:react`)
```json
{
  "event": "chat:react",
  "payload": {
    "message_id": "uuid-msg-1111",
    "emoji": "🔥",
    "action": "add" // add hoặc remove
  }
}
```

---

### Các Sự kiện WebSocket Server Phát (Server -> Client)

#### 1. Nhận tin nhắn mới (`chat:message_received`)
Broadcast tới toàn bộ client trong phòng khi có tin nhắn mới.
```json
{
  "event": "chat:message_received",
  "payload": {
    "id": "uuid-msg-2222",
    "sender": {
      "id": "uuid-user-9999",
      "username": "alice",
      "avatar_url": "https://..."
    },
    "content": "Hello world! @john_doe",
    "reply_to_id": null,
    "created_at": "2026-06-07T13:14:00Z"
  }
}
```

#### 2. Broadcast trạng thái soạn thảo (`chat:member_typing`)
```json
{
  "event": "chat:member_typing",
  "payload": {
    "user_id": "uuid-user-9999",
    "username": "alice",
    "is_typing": true
  }
}
```

#### 3. Cập nhật lượt tương tác (`chat:reaction_updated`)
```json
{
  "event": "chat:reaction_updated",
  "payload": {
    "message_id": "uuid-msg-1111",
    "emoji": "🔥",
    "user_id": "uuid-user-9999",
    "action": "added" // added hoặc removed
  }
}
```

---

## 2. Các API RESTful

### 2.1 Lấy lịch sử tin nhắn
Phục vụ tải tin nhắn cũ khi mở phòng hoặc cuộn vô tận (infinite scroll).

*   **Endpoint:** `GET /messages`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `before_id`: Lấy tin nhắn được gửi trước ID tin nhắn này (phân trang).
    *   `limit`: Giới hạn số lượng tin nhắn (mặc định 50).
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "id": "uuid-msg-1111",
          "sender_id": "uuid-user-9999",
          "content": "Tin nhắn cũ hơn",
          "reply_to_id": null,
          "created_at": "2026-06-07T13:00:00Z"
        }
      ],
      "error": null
    }
    ```

### 2.2 Sửa tin nhắn
Cho phép chỉnh sửa nội dung tin nhắn trong vòng 15 phút từ lúc gửi.

*   **Endpoint:** `PUT /messages/:msg_id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "content": "Nội dung tin nhắn đã được chỉnh sửa"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "id": "uuid-msg-1111",
        "content": "Nội dung tin nhắn đã được chỉnh sửa",
        "is_edited": true,
        "updated_at": "2026-06-07T13:15:00Z"
      },
      "error": null
    }
    ```

### 2.3 Thu hồi tin nhắn
Xóa bỏ tin nhắn của mình hoặc của người khác (nếu có quyền Moderator).

*   **Endpoint:** `DELETE /messages/:msg_id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã thu hồi tin nhắn thành công."
      },
      "error": null
    }
    ```
