# User Service API Specification

Dịch vụ User quản lý hồ sơ người dùng (UserProfile), trạng thái hoạt động trực tuyến (Presence) và danh sách bạn bè (Friends).

## Base URL
`/api/v1/users`

---

## 1. Xem hồ sơ cá nhân
Lấy thông tin công khai của bất kỳ người dùng nào qua ID.

*   **Endpoint:** `GET /profile/:id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "id": "uuid-user-1234",
        "username": "johndoe",
        "display_name": "John Doe",
        "avatar_url": "https://minio/avatar/johndoe.png",
        "bio": "Keep coding and listening to music.",
        "presence": {
          "status": "online",
          "custom_text": "Listening to Lofi",
          "last_active": 1780824000
        },
        "created_at": "2026-06-01T00:00:00Z"
      },
      "error": null
    }
    ```

---

## 2. Cập nhật hồ sơ cá nhân
Thay đổi thông tin hiển thị của bản thân.

*   **Endpoint:** `PUT /profile`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "display_name": "John Updated",
      "bio": "A passionate software developer.",
      "avatar_url": "https://minio/avatar/new_johndoe.png"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "id": "uuid-user-1234",
        "display_name": "John Updated",
        "bio": "A passionate software developer.",
        "avatar_url": "https://minio/avatar/new_johndoe.png"
      },
      "error": null
    }
    ```

---

## 3. Cập nhật Trạng thái hoạt động (Presence)
Cập nhật trạng thái trực tuyến hiển thị cho bạn bè hoặc các thành viên trong phòng cùng thấy.

*   **Endpoint:** `PUT /status`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "status": "busy", // online, offline, busy, away
      "custom_text": "Đang tập trung học bài"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "status": "busy",
        "custom_text": "Đang tập trung học bài",
        "updated_at": "2026-06-07T13:10:00Z"
      },
      "error": null
    }
    ```

---

## 4. Quản lý bạn bè (Friends)

### 4.1 Gửi lời mời kết bạn
*   **Endpoint:** `POST /friends/request`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "friend_id": "uuid-friend-5678"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "id": "friendship-uuid-9999",
        "status": "PENDING",
        "message": "Đã gửi yêu cầu kết bạn."
      },
      "error": null
    }
    ```

### 4.2 Lấy danh sách bạn bè
*   **Endpoint:** `GET /friends`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `status`: `ACCEPTED` | `PENDING` (chờ duyệt) | `BLOCKED` (bị chặn)
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "friendship_id": "friendship-uuid-9999",
          "friend_profile": {
            "id": "uuid-friend-5678",
            "username": "friend_user",
            "display_name": "My Friend",
            "avatar_url": "https://..."
          },
          "status": "ACCEPTED",
          "updated_at": "2026-06-07T12:00:00Z"
        }
      ],
      "error": null
    }
    ```

### 4.3 Phản hồi lời mời kết bạn
*   **Endpoint:** `PUT /friends/request/:id`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "action": "accept" // accept hoặc reject
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã chấp nhận lời mời kết bạn."
      },
      "error": null
    }
    ```

### 4.4 Chặn người dùng
*   **Endpoint:** `POST /friends/block`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "target_id": "uuid-user-to-block"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã chặn người dùng thành công."
      },
      "error": null
    }
    ```
