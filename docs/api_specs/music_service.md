# Music Service API Specification

Dịch vụ Music chịu trách nhiệm tìm kiếm bài hát từ các nguồn mở (YouTube trong MVP), trích xuất metadata (Tiêu đề, Nghệ sĩ, Độ dài, Thumbnail), quản lý upload file âm thanh lên MinIO, và ghi nhận lịch sử phát.

## Base URL
`/api/v1/music`

---

## 1. Tìm kiếm và Trích xuất Metadata YouTube
Tìm kiếm từ khóa hoặc phân tích trực tiếp một URL của YouTube để trích xuất metadata của bài hát.

*   **Endpoint:** `GET /search`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `query`: Từ khóa tìm kiếm hoặc đường link đầy đủ của YouTube (ví dụ: `https://www.youtube.com/watch?v=...` hoặc `lofi hip hop`).
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "title": "Lofi Chill Beats to Study/Relax",
          "artist": "Lofi Records",
          "duration_ms": 180000,
          "thumbnail_url": "https://img.youtube.com/vi/abc123xyz/hqdefault.jpg",
          "source_type": "YOUTUBE",
          "source_url": "https://www.youtube.com/watch?v=abc123xyz"
        }
      ],
      "error": null
    }
    ```

---

## 2. Upload file âm thanh
Hỗ trợ người dùng tải trực tiếp file nhạc cá nhân (`mp3`, `wav`, `flac`) lên MinIO Storage.

*   **Endpoint:** `POST /upload`
*   **Headers:**
    *   `Authorization: Bearer <access_token>`
    *   `Content-Type: multipart/form-data`
*   **Request Body (Form Data):**
    *   `file`: File nhị phân (giới hạn 50MB).
*   **Response (201 Created):**
    ```json
    {
      "success": true,
      "data": {
        "track_id": "uuid-track-9999",
        "title": "My Favorite Song",
        "artist": "Unknown Artist",
        "duration_ms": 240000,
        "thumbnail_url": "https://minio/bucket/default_music.png",
        "source_type": "UPLOAD",
        "source_url": "https://minio/bucket/tracks/uuid-track-9999.mp3"
      },
      "error": null
    }
    ```

---

## 3. Lịch sử phát nhạc trong phòng
Lấy danh sách các bài hát đã phát gần đây trong một phòng cụ thể.

*   **Endpoint:** `GET /rooms/:room_id/history`
*   **Headers:** `Authorization: Bearer <access_token>`
*   **Query Parameters:**
    *   `limit`: Số lượng bản ghi (mặc định 20).
*   **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "history_id": "uuid-hist-1111",
          "track": {
            "id": "uuid-track-9999",
            "title": "Lofi Chill Beats",
            "artist": "Lofi Records"
          },
          "played_at": "2026-06-07T13:00:00Z"
        }
      ],
      "error": null
    }
    ```
