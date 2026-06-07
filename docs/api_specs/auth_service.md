# Auth Service API Specification

Dịch vụ Authentication chịu trách nhiệm đăng ký, xác thực người dùng, cấp phát JWT và duy trì phiên đăng nhập.

## Base URL
`/api/v1/auth`

---

## 1. Đăng ký tài khoản
Đăng ký tài khoản người dùng mới. Sau khi đăng ký thành công, một email chứa mã xác thực sẽ được gửi đi.

*   **Endpoint:** `POST /register`
*   **Request Body:**
    ```json
    {
      "email": "user@example.com",
      "username": "user123",
      "password": "Password123"
    }
    ```
*   **Validation Rules:**
    *   `email`: Phải là địa chỉ email hợp lệ và chưa tồn tại trong hệ thống.
    *   `username`: 3-30 ký tự, chỉ chứa ký tự chữ, số và dấu gạch dưới `_`.
    *   `password`: Tối thiểu 8 ký tự, có ít nhất 1 chữ hoa, 1 chữ thường và 1 chữ số.
*   **Response (201 Created):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đăng ký thành công. Vui lòng kiểm tra email để kích hoạt tài khoản."
      },
      "error": null
    }
    ```
*   **Response (400 Bad Request):**
    ```json
    {
      "success": false,
      "data": null,
      "error": {
        "code": "EMAIL_ALREADY_EXISTS",
        "message": "Email này đã được đăng ký sử dụng."
      }
    }
    ```

---

## 2. Xác thực Email
Kích hoạt tài khoản bằng mã xác nhận gửi qua email.

*   **Endpoint:** `POST /verify-email`
*   **Request Body:**
    ```json
    {
      "token": "verification_token_string_here"
    }
    ```
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Tài khoản đã được xác thực thành công."
      },
      "error": null
    }
    ```

---

## 3. Đăng nhập
Xác thực người dùng và nhận Access Token (JWT) và Refresh Token (Cookie).

*   **Endpoint:** `POST /login`
*   **Request Body:**
    ```json
    {
      "identity": "user123_or_email",
      "password": "Password123"
    }
    ```
*   **Headers:**
    *   `X-Forwarded-For` / `User-Agent`: Để ghi nhận vị trí session.
*   **Response Headers:**
    *   `Set-Cookie: refresh_token=ref_abc123...; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict; Max-Age=604800`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "access_token": "jwt_access_token_here",
        "expires_in": 900
      },
      "error": null
    }
    ```

---

## 4. Đăng xuất
Thu hồi Refresh Token hiện tại và xóa phiên đăng nhập.

*   **Endpoint:** `POST /logout`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Cookies:** `refresh_token=<token>`
*   **Response Headers:**
    *   `Set-Cookie: refresh_token=; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "message": "Đăng xuất thành công."
      },
      "error": null
    }
    ```

---

## 5. Làm mới Token (Refresh Token Rotation)
Sử dụng Refresh Token hợp lệ để lấy Access Token mới và xoay vòng Refresh Token cũ.

*   **Endpoint:** `POST /refresh`
*   **Cookies:** `refresh_token=<token>`
*   **Response Headers:**
    *   `Set-Cookie: refresh_token=new_ref_token...; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict`
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "access_token": "new_jwt_access_token_here",
        "expires_in": 900
      },
      "error": null
    }
    ```

---

## 6. Đăng nhập Google (OAuth2 với PKCE)
Bắt đầu luồng xác thực Google OAuth2.

*   **Endpoint:** `GET /oauth/google`
*   **Query Parameters:**
    *   `code_challenge`: Chuỗi SHA256 được mã hóa Base64URL.
    *   `redirect_uri`: Địa chỉ Client hứng Callback.
*   **Response:** Redirect 302 đến trang đăng nhập Google.

### OAuth2 Callback
Cổng callback nhận mã code và trao đổi token.
*   **Endpoint:** `POST /oauth/google/callback`
*   **Request Body:**
    ```json
    {
      "code": "auth_code_from_google",
      "code_verifier": "original_random_string_before_hash"
    }
    ```
*   **Response:** Trả về Access Token & đặt Refresh Token Cookie tương tự API Login.
