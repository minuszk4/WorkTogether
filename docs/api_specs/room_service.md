# Room Service API Specification

Dịch vụ Room quản lý việc tạo, chỉnh sửa phòng cộng tác, tham gia/rời phòng, và phân quyền chi tiết (Custom Roles) trong từng phòng.

## Base URL
`/api/v1/rooms`

---

## 1. Tạo phòng mới
*   **Endpoint:** `POST /`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "name": "Lounge Coder",
      "description": "Nghe nhạc và lập trình Go cùng nhau",
      "privacy": "private", // public, private, friends
      "password": "optional_room_password"
    }
    ```
*   **Response (201 Created):**
    ```json
    {
      "success": true,
      "data": {
        "id": "uuid-room-1111",
        "name": "Lounge Coder",
        "description": "Nghe nhạc và lập trình Go cùng nhau",
        "privacy": "private",
        "invite_code": "LOUNGE1",
        "owner_id": "uuid-user-9999",
        "created_at": "2026-06-07T13:12:00Z"
      },
      "error": null
    }
    ```

---

## 2. Tìm kiếm / Lấy danh sách phòng
Lấy danh sách các phòng công khai (public).

*   **Endpoint:** `GET /`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `search`: Từ khóa tìm kiếm theo tên phòng.
    *   `limit`: Giới hạn số kết quả (mặc định 20).
    *   `offset`: Phân trang.
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "id": "uuid-room-1111",
          "name": "Lounge Coder",
          "description": "Nghe nhạc và lập trình Go cùng nhau",
          "privacy": "public",
          "active_members": 8,
          "owner_id": "uuid-user-9999"
        }
      ],
      "error": null
    }
    ```

---

## 3. Tham gia phòng
*   **Endpoint:** `POST /:id/join`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "password": "optional_room_password"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "room_id": "uuid-room-1111",
        "role": "MEMBER",
        "permissions": ["CAN_CHAT", "CAN_ADD_MUSIC"]
      },
      "error": null
    }
    ```

---

## 4. Quản lý phân quyền chi tiết (Custom Roles)

Nhằm đáp ứng yêu cầu linh hoạt về phân quyền trong phòng, chủ phòng (Owner) hoặc Moderator có quyền quản lý vai trò có thể tạo ra các Role tùy chỉnh với các cờ phân quyền cụ thể.

### 4.1 Tạo Role mới trong phòng
*   **Endpoint:** `POST /:id/roles`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "name": "DJ Room",
      "permissions": {
        "can_chat": true,
        "can_manage_playlist": true,
        "can_control_playback": true,
        "can_moderate_members": false,
        "can_use_voice": true
      }
    }
    ```
*   **Response (201 Created):**
    ```json
    {
      "success": true,
      "data": {
        "role_id": "uuid-role-dj",
        "name": "DJ Room",
        "permissions": {
          "can_chat": true,
          "can_manage_playlist": true,
          "can_control_playback": true,
          "can_moderate_members": false,
          "can_use_voice": true
        }
      },
      "error": null
    }
    ```

### 4.2 Gán Role cho thành viên
*   **Endpoint:** `PUT /:id/members/:user_id/role`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "role_id": "uuid-role-dj"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã gán vai trò DJ Room cho thành viên."
      },
      "error": null
    }
    ```

---

## 5. Xử lý vi phạm thành viên (Kick, Ban, Mute)

### 5.1 Kick thành viên (Trục xuất tạm thời)
Trục xuất thành viên khỏi phòng. Người dùng có thể quay lại phòng nếu có quyền truy cập.

*   **Endpoint:** `POST /:id/members/:user_id/kick`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã kick thành viên khỏi phòng."
      },
      "error": null
    }
    ```

### 5.2 Ban thành viên (Cấm vĩnh viễn)
Cấm vĩnh viễn thành viên khỏi phòng, đồng thời ghi nhận vào danh sách đen `room_bans`.

*   **Endpoint:** `POST /:id/members/:user_id/ban`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "reason": "Spam liên tục trong phòng"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã ban thành viên khỏi phòng."
      },
      "error": null
    }
    ```

### 5.3 Mute thành viên (Tắt quyền chat)
Tạm thời thu hồi quyền chat (`can_chat = false`) của thành viên trong phòng.

*   **Endpoint:** `POST /:id/members/:user_id/mute`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Request Body:**
    ```json
    {
      "duration_seconds": 600 // Mute trong 10 phút
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đã tắt quyền chat của thành viên trong 10 phút."
      },
      "error": null
    }
    ```
