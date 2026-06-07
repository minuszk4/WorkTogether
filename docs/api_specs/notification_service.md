# Notification Service API Specification

Dịch vụ Notification chịu trách nhiệm lưu trữ và đẩy các thông báo (như lời mời kết bạn, được nhắc tên trong phòng chat, thông báo phát nhạc, cuộc gọi thoại) đến người dùng thời gian thực qua giao thức Server-Sent Events (SSE).

## Base URL
`/api/v1/notifications`

---

## 1. Lấy danh sách thông báo (Pull API)
Lấy lịch sử thông báo của người dùng hiện tại để hiển thị trên UI panel.

*   **Endpoint:** `GET /`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `limit`: Số lượng thông báo (mặc định 20).
    *   `unread_only`: Lọc chỉ lấy các thông báo chưa đọc (`true`/`false`).
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "id": "uuid-noti-1111",
          "user_id": "uuid-user-9999",
          "type": "FRIEND_REQUEST", // FRIEND_REQUEST, CHAT_MENTION, ROOM_INVITE
          "title": "Lời mời kết bạn",
          "content": "Nguyễn Văn B đã gửi lời mời kết bạn với bạn.",
          "is_read": false,
          "sender_id": "uuid-user-5555",
          "created_at": "2026-06-07T13:10:00Z"
        }
      ],
      "error": null
    }
    ```

---

## 2. Đánh dấu đã đọc thông báo
*   **Endpoint:** `PUT /:id/read`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã đánh dấu đọc thông báo thành công."
      },
      "error": null
    }
    ```

---

## 3. Xóa thông báo
*   **Endpoint:** `DELETE /:id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã xóa thông báo."
      },
      "error": null
    }
    ```

---

## 4. Real-time Notification Stream (SSE)
Client đăng ký kết nối Server-Sent Events (SSE) để nhận các thông báo đẩy ngay lập tức mà không cần gọi API liên tục (Polling).

*   **Endpoint:** `GET /stream`
*   **Headers:**
    *   `Authorization: Bearer <access_token>`
    *   `Accept: text/event-stream`
    *   `Cache-Control: no-cache`
    *   `Connection: keep-alive`

### Mẫu Event dữ liệu nhận được từ Stream (SSE Format)
```
event: notification
data: {"id":"uuid-noti-1111","type":"CHAT_MENTION","title":"Bạn được nhắc tới","content":"alice đã nhắc tới bạn trong phòng Lounge Coder","created_at":"2026-06-07T13:16:00Z"}
```
*(Ghi chú: Kết nối được duy trì lâu dài, Server gửi định kỳ tin nhắn trống `ping` mỗi 30s để tránh bị các Proxy/Gateways đóng kết nối).*
