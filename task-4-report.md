# 📋 Báo cáo Sửa đổi Task 4 - Chat Service

Dưới đây là chi tiết các thay đổi và kết quả sửa lỗi trong Task 4 liên quan đến bảo mật tin nhắn chéo phòng (cross-room tampering), giới hạn ghim tin nhắn, cập nhật WebSocket handlers, và xác thực quyền thành viên trong REST endpoints.

---

## 🛠️ Chi tiết các thay đổi

### 1. Sửa lỗi bảo mật tin nhắn chéo phòng & Giới hạn ghim tin nhắn (Global Unauthorized Unpinning & Cross-Room Message Tampering)
- **Tập tin:** [services/chat-service/internal/repository/postgres.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/repository/postgres.go)
  - Thêm phương thức `GetPinnedCount(ctx, roomID)` để đếm số lượng tin nhắn đã được ghim trong một phòng cụ thể.
- **Tập tin:** [services/chat-service/internal/usecase/chat.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/usecase/chat.go)
  - Cập nhật các phương thức `EditMessage`, `DeleteMessage`, `AddReaction`, `RemoveReaction`, `PinMessage`, và `UnpinMessage` để thực hiện kiểm tra `RoomID` của tin nhắn khớp với `roomID` yêu cầu (tránh giả mạo chéo phòng). Nếu không khớp hoặc tin nhắn không tồn tại, trả về `ErrUnauthorized`.
  - Trong `PinMessage`, kiểm tra `GetPinnedCount` và chặn không cho phép ghim nếu số tin nhắn đã ghim trong phòng `>= 10` (trả về lỗi `"vượt quá giới hạn 10 tin nhắn ghim cho phòng này"`).

### 2. Cập nhật WS Handler (WS Handler Updates)
- **Tập tin:** [services/chat-service/internal/delivery/http/handlers.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/delivery/http/handlers.go)
  - Cập nhật hàm `readPump` xử lý các WebSocket event sau để truyền thêm `RoomID` hoặc tham số phù hợp vào tầng Usecase:
    - Event `chat:edit_message`: truyền `c.RoomID`.
    - Event `chat:delete_message`: truyền `c.RoomID`.
    - Event `chat:react`: truyền `c.RoomID`.
    - Event `chat:unpin_message`: truyền `c.UserID` và `c.RoomID`.

### 3. Bổ sung kiểm tra thành viên phòng chat trong REST Endpoints
- **Tập tin:** [services/chat-service/internal/delivery/http/handlers.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/delivery/http/handlers.go)
  - Trong `GetMessages` và `SearchMessages`, truy vấn `VerifyRoomMember` từ `roomClient` để xác thực người dùng hiện tại (`userID := c.GetString("userID")`) là thành viên của `roomID`. Nếu không phải, trả về HTTP status `403 Forbidden` với định dạng JSON lỗi chuẩn:
    ```json
    {
      "success": false,
      "data": null,
      "error": {
        "code": "FORBIDDEN",
        "message": "Bạn không phải thành viên của phòng này."
      }
    }
    ```

### 4. Code Quality & Best Practices
- **Tập tin:** [services/chat-service/internal/repository/postgres.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/repository/postgres.go)
  - Bổ sung kiểm tra lỗi sau vòng lặp lặp hàng trong `SearchMessages` bằng cách gọi `rows.Err()`.
- **Tập tin:** [services/chat-service/internal/usecase/chat_test.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/chat-service/internal/usecase/chat_test.go)
  - Tạo mới bộ unit test đầy đủ bao quát logic sửa đổi của `EditMessage` và `PinMessage` (sử dụng thư viện `go-sqlmock` để giả lập database).

---

## 💾 Thông tin Commit
- **Commit hash:** `6c52b29`
- **Thông điệp:** `feat(chat): fix cross-room validation, pin limit, and member verification in HTTP/WS handlers`

---

## 🧪 Kết quả Kiểm thử (Verification)

Bộ kiểm thử của `chat-service` được biên dịch và thực thi thành công vượt qua tất cả các test cases:

```bash
$ go test -v ./internal/usecase
=== RUN   TestEditMessage
=== RUN   TestEditMessage/Success
=== RUN   TestEditMessage/Cross-Room_Message_Tampering_Rejected
=== RUN   TestEditMessage/Unauthorized_Edit
=== RUN   TestEditMessage/Edit_Time_Expired
--- PASS: TestEditMessage (0.00s)
    --- PASS: TestEditMessage/Success (0.00s)
    --- PASS: TestEditMessage/Cross-Room_Message_Tampering_Rejected (0.00s)
    --- PASS: TestEditMessage/Unauthorized_Edit (0.00s)
    --- PASS: TestEditMessage/Edit_Time_Expired (0.00s)
=== RUN   TestPinMessage
=== RUN   TestPinMessage/Success
=== RUN   TestPinMessage/Cross-Room_Pin_Rejected
=== RUN   TestPinMessage/Pin_Limit_Exceeded
--- PASS: TestPinMessage (0.00s)
    --- PASS: TestPinMessage/Success (0.00s)
    --- PASS: TestPinMessage/Cross-Room_Pin_Rejected (0.00s)
    --- PASS: TestPinMessage/Pin_Limit_Exceeded (0.00s)
PASS
ok  	github.com/worktogether/services/chat-service/internal/usecase	1.061s
```

Chạy toàn bộ test của chat service:
```bash
$ go test ./...
ok  	github.com/worktogether/services/chat-service/internal/delivery/http	(cached)
ok  	github.com/worktogether/services/chat-service/internal/usecase	1.061s
```
