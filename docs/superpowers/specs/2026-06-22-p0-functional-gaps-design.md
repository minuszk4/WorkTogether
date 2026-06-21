# Specification: P0 Functional Gaps — Technical Design

**Tạo:** 2026-06-22  
**Trạng thái:** Approved  
**Tác giả:** Antigravity  

---

## 1. Goal
Tài liệu thiết kế chi tiết để giải quyết các thiếu sót chức năng nghiêm trọng thuộc mức độ **P0 (Must-Have)** được phát hiện qua báo cáo review:
1. **Authentication (auth-service)**: Bổ sung chức năng Quên mật khẩu, Đặt lại mật khẩu và Đổi mật khẩu.
2. **Room (room-service)**: Bổ sung chức năng Xóa phòng và Mute thành viên (cấm chat/voice tạm thời).
3. **Chat (chat-service)**: Bổ sung các sự kiện WebSocket để Sửa, Thu hồi/Xóa, Pin/Unpin tin nhắn; thêm API tìm kiếm tin nhắn; và tự động nhận diện `@mention` để trigger thông báo qua Redis Stream.

---

## 2. Authentication Service (`auth-service`)

### 2.1 API Endpoints
Bổ sung các endpoints sau vào router của `auth-service`:
- **`POST /api/v1/auth/forgot-password`** (Public)
  - Yêu cầu body: `{ "email": "user@example.com" }`
  - Logic: Kiểm tra tài khoản có email này. Sinh JWT token reset password có TTL 15 phút chứa claims: `{"sub": userID, "type": "password_reset"}`. Gửi email chứa link reset password: `${FRONTEND_URL}/auth/reset-password?token=${token}`.
- **`POST /api/v1/auth/reset-password`** (Public)
  - Yêu cầu body: `{ "token": "...", "new_password": "..." }`
  - Logic: Giải mã & validate reset token. Hash mật khẩu mới và cập nhật trong bảng `accounts`.
- **`POST /api/v1/auth/change-password`** (Bảo vệ bởi `AuthMiddleware`)
  - Yêu cầu body: `{ "old_password": "...", "new_password": "..." }`
  - Logic: Lấy `userID` từ context. Xác thực `old_password` có khớp với hash hiện tại không. Hash `new_password` mới và cập nhật.

### 2.2 Email Service
Bổ sung phương thức gửi mail reset password:
```go
func (e *EmailService) SendPasswordResetEmail(toEmail, username, resetURL string) error {
    if !e.IsConfigured() {
        fmt.Printf("[EMAIL-DEV] Gửi link reset mật khẩu tới %s: %s\n", toEmail, resetURL)
        return nil
    }
    subject := "WorkTogether – Khôi phục mật khẩu của bạn"
    body := buildResetEmailHTML(username, resetURL)
    return e.sendHTML(toEmail, subject, body)
}
```

### 2.3 Middleware
Sao chép `AuthMiddleware` chuẩn (đang dùng ở các service khác) vào `auth-service/pkg/middleware/auth.go` để bảo vệ API `/change-password`.

---

## 3. Room Service (`room-service`)

### 3.1 Database Schema Migration
Tạo migration `000005_add_mute_to_members.up.sql` để thêm cột `muted_until`:
```sql
ALTER TABLE room_members ADD COLUMN IF NOT EXISTS muted_until TIMESTAMP WITH TIME ZONE;
```

### 3.2 standalone Room Deletion
Expose endpoint: **`DELETE /api/v1/rooms/:id`**
- **Logic**: Kiểm tra người thực hiện request có phải là OWNER của phòng không (`role_type = 'OWNER'`). Nếu đúng, tiến hành xóa phòng khỏi database và trả về 200 OK.

### 3.3 Member Muting
Bổ sung các API quản lý trạng thái cấm nói/chat:
- **`POST /api/v1/rooms/:id/members/:user_id/mute`**
  - Body: `{ "duration_seconds": 300 }`
  - Logic: Kiểm tra quyền của người gọi (phải là OWNER hoặc MODERATOR). Kiểm tra đối tượng bị mute (không được là OWNER). Tính toán `muted_until = time.Now().Add(duration)`. Lưu vào bảng `room_members`.
- **`POST /api/v1/rooms/:id/members/:user_id/unmute`**
  - Logic: Xóa trạng thái mute bằng cách set `muted_until = NULL`.

### 3.4 Cập nhật gRPC `VerifyRoomMember`
Khi biên dịch các permissions của user để trả về cho các service khác:
- Nếu `muted_until != nil` và `muted_until > time.Now()`, Server sẽ loại bỏ quyền `"CAN_CHAT"` và `"CAN_USE_VOICE"` khỏi mảng `permissions` trước khi phản hồi.

---

## 4. Chat Service (`chat-service`)

### 4.1 Sự kiện WebSocket quản lý tin nhắn
Trong switch-case của `readPump` thuộc WebSocket handler:
- **`chat:edit_message`**:
  - Payload: `{ "message_id": "...", "content": "..." }`
  - Xử lý: Gọi `uc.EditMessage(...)`. Nếu thành công, broadcast sự kiện `chat:message_edited` chứa `{ "message_id": "...", "content": "...", "is_edited": true }`.
- **`chat:delete_message`**:
  - Payload: `{ "message_id": "..." }`
  - Xử lý: Gọi `uc.DeleteMessage(...)` (cho phép người gửi tự xóa, hoặc moderator có quyền xóa bất kỳ). Nếu thành công, broadcast sự kiện `chat:message_deleted` chứa `{ "message_id": "..." }`.
- **`chat:pin_message`** & **`chat:unpin_message`**:
  - Tương tác với bảng `message_pins` và broadcast sự kiện `chat:message_pinned`/`chat:message_unpinned`. Kiểm tra giới hạn 10 tin nhắn pin tối đa cho một phòng.

### 4.2 REST API: Tìm kiếm tin nhắn
Expose endpoint: **`GET /api/v1/rooms/:id/chat/search?q=query`**
- **Logic**: Sử dụng mệnh đề `ILIKE` trên cột `content` lọc theo `room_id`. Trả về danh sách tin nhắn khớp.

### 4.3 Nhận diện `@mention` và gửi Notification
Trong payload của sự kiện gửi tin nhắn `chat:send_message`, frontend sẽ đính kèm danh sách `mentions` chứa user ID của các người dùng được nhắc đến (được tự động hoàn thành từ UI):
- Payload `chat:send_message` cập nhật:
  ```json
  {
    "content": "Chào bạn @Alice và @Bob",
    "reply_to_id": "",
    "mentions": ["uuid-alice-123", "uuid-bob-456"]
  }
  ```
- **Xử lý trên Backend**: Với mỗi user ID trong `mentions`, `chat-service` sử dụng Redis Client để `XAdd` một thông báo trigger vào Redis Stream `stream:notification_trigger` để `notification-service` xử lý không đồng bộ:
  ```json
  {
    "receiver_id": "uuid-alice-123",
    "sender_id": "uuid-sender-789",
    "type": "mention",
    "content": "bạn được nhắc đến trong phòng."
  }
  ```

---

## 5. Verification Plan
- **Backend Unit Tests**: Viết kiểm thử tự động cho các repository và usecase mới.
- **API Verification**: Sử dụng curl để verify các route RESTful mới hoạt động đúng.
