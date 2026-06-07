# SOFTWARE REQUIREMENTS SPECIFICATION (SRS)

# Dự án: SyncSpace

**Phiên bản:** 1.1
**Tác giả:** Nhóm phát triển
**Ngày cập nhật:** 07/06/2026
**Kiến trúc:** Golang Microservices + PostgreSQL

---

## MỤC LỤC

1. Giới thiệu
2. Phạm vi hệ thống
3. Kiến trúc Microservices
4. Yêu cầu chức năng
5. Yêu cầu phi chức năng
6. Yêu cầu giao diện người dùng (UI/UX)
7. Phiên bản MVP

---

# 1. GIỚI THIỆU

## 1.1 Mục đích

SyncSpace là nền tảng cộng tác thời gian thực cho phép nhiều người dùng tham gia cùng một không gian trực tuyến để:

- Nghe nhạc cùng nhau (đồng bộ playback)
- Trò chuyện thời gian thực (text chat)
- Voice Call & Video Call
- Chia sẻ màn hình
- Học nhóm & làm việc nhóm
- Quản lý playlist cộng tác

Hệ thống tập trung vào đồng bộ trạng thái giữa nhiều người dùng trong cùng một phòng với độ trễ thấp và khả năng mở rộng cao, được xây dựng theo kiến trúc **Golang Microservices**.

---

## 1.2 Đối tượng sử dụng

| Nhóm | Nhu cầu chính |
|------|---------------|
| Sinh viên học nhóm | Phòng học, chia sẻ màn hình, nhạc nền |
| Nhóm làm việc từ xa | Voice/Video call, screen share, chat |
| Cộng đồng nghe nhạc | Playlist cộng tác, vote bài hát |
| Nhóm lập trình | Screen share, code session |
| Người dùng cá nhân | Nghe nhạc, kết bạn |

---

## 1.3 Công nghệ dự kiến

### Backend (Microservices - Golang)

| Service | Công nghệ |
|---------|-----------|
| API Gateway | Golang + Gin/Fiber, JWT middleware |
| Auth Service | Golang, OAuth2, bcrypt |
| User Service | Golang + sqlc/pgx |
| Room Service | Golang + WebSocket |
| Chat Service | Golang + WebSocket |
| Music Service | Golang |
| Playback Service | Golang + Redis Pub/Sub |
| Voice/Video Service | Golang + LiveKit SDK |
| Notification Service | Golang |
| Search Service | Golang + PostgreSQL FTS |
| Analytics Service | Golang |
| Admin Service | Golang |

### Database & Storage

| Thành phần | Công nghệ | Mục đích |
|------------|-----------|----------|
| Primary DB | PostgreSQL 15+ | Dữ liệu chính, mỗi service có schema riêng |
| Cache | Redis 7+ | Session, Pub/Sub, rate limiting, playback state |
| File Storage | MinIO | Upload âm nhạc, avatar, media |
| Message Queue | Redis Streams / NATS | Giao tiếp giữa các service |

### Infrastructure

| Thành phần | Công nghệ |
|------------|-----------|
| Realtime | WebSocket (gorilla/websocket) |
| Voice/Video | LiveKit |
| Service Discovery | Kubernetes DNS |
| API Gateway | Golang tự build hoặc Kong |
| Monitoring | Prometheus + Grafana + Jaeger |
| Deployment | Docker + Kubernetes (K8s) |
| CI/CD | GitHub Actions |
| Logging | Loki + Grafana |

---

# 2. PHẠM VI HỆ THỐNG

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway                          │
│             (Auth, Rate Limit, Routing)                 │
└──────────┬──────────────────────────────┬──────────────┘
           │                              │
    ┌──────▼──────┐                ┌──────▼──────┐
    │ REST APIs   │                │  WebSocket  │
    │ (HTTP/2)    │                │  Gateway    │
    └──────┬──────┘                └──────┬──────┘
           │                              │
    ┌──────▼──────────────────────────────▼──────┐
    │              Microservices Layer            │
    │  auth │ user │ room │ chat │ music │ ...    │
    └──────────────────┬──────────────────────────┘
                       │
    ┌──────────────────▼──────────────────────────┐
    │           Data Layer                        │
    │   PostgreSQL │ Redis │ MinIO                │
    └─────────────────────────────────────────────┘
```

---

# 3. KIẾN TRÚC MICROSERVICES

## 3.1 Danh sách Services

| Service | Port | Trách nhiệm |
|---------|------|-------------|
| api-gateway | 8080 | Routing, Auth middleware, Rate limit |
| auth-service | 8081 | Login, Register, Token |
| user-service | 8082 | Profile, Friends, Presence |
| room-service | 8083 | CRUD Room, Members, Roles |
| chat-service | 8084 | Messages, WebSocket Hub |
| music-service | 8085 | Sources, Metadata, Upload |
| playlist-service | 8086 | Playlist CRUD, Queue |
| playback-service | 8087 | Sync State, Redis Pub/Sub |
| voice-service | 8088 | LiveKit integration |
| notification-service | 8089 | Push, In-app noti |
| search-service | 8090 | Full-text search |
| analytics-service | 8091 | Metrics, Reports |
| admin-service | 8092 | Admin dashboard |

## 3.2 Giao tiếp giữa Services

- **Đồng bộ (Synchronous):** REST/gRPC qua internal network
- **Bất đồng bộ (Asynchronous):** Redis Streams hoặc NATS cho event-driven
- **Realtime:** WebSocket tập trung tại chat-service và playback-service

## 3.3 Database per Service

Mỗi microservice có **schema riêng trong PostgreSQL** hoặc database riêng biệt (tùy scale):

```
syncspace_auth     → users, sessions, oauth_tokens
syncspace_user     → profiles, friends, blocks
syncspace_room     → rooms, members, roles, bans
syncspace_chat     → messages, reactions, pins
syncspace_music    → tracks, sources, history
syncspace_playlist → playlists, playlist_tracks, votes
syncspace_notification → notifications
syncspace_analytics   → events, aggregations
```

---

# 4. YÊU CẦU CHỨC NĂNG

## MODULE AUTHENTICATION (auth-service)

### AUTH-01 Đăng ký tài khoản

**Input:** Email, Username, Password
**Output:** Account Created, Verification Email gửi đi
**Validation:**
- Email hợp lệ, duy nhất trong hệ thống
- Username: 3-30 ký tự, alphanumeric + underscore, duy nhất
- Password: tối thiểu 8 ký tự, phải có chữ hoa, chữ thường, số

### AUTH-02 Xác thực email

- Gửi email với verification token (TTL: 24h)
- Click link → kích hoạt tài khoản
- Có thể gửi lại email xác thực

### AUTH-03 Đăng nhập

**Input:** Email/Username + Password
**Output:** Access Token (JWT, TTL: 15 phút), Refresh Token (TTL: 7 ngày)
**Error cases:** Sai mật khẩu, chưa verify email, tài khoản bị khóa

### AUTH-04 Đăng xuất

- Thu hồi Refresh Token (blacklist vào Redis)
- Xóa session

### AUTH-05 Làm mới token

- Dùng Refresh Token hợp lệ → sinh Access Token mới
- Rotate Refresh Token (invalidate token cũ)

### AUTH-06 Quên mật khẩu

- Gửi email reset link (TTL: 1h, single-use)

### AUTH-07 Đặt lại mật khẩu

- Validate reset token → cho phép nhập mật khẩu mới
- Invalidate tất cả Refresh Token hiện có của user

### AUTH-08 Đổi mật khẩu

**Yêu cầu:** Mật khẩu cũ đúng + mật khẩu mới thỏa validation

### AUTH-09 Đăng nhập Google (OAuth2)

- Callback URL, lấy thông tin profile từ Google
- Tự tạo tài khoản nếu email chưa tồn tại

---

## MODULE USER (user-service)

### USER-01 Xem hồ sơ cá nhân

Hiển thị: Avatar, Username, Display Name, Bio, Trạng thái, Ngày tham gia, Số bạn bè

### USER-02 Cập nhật hồ sơ

Cho phép sửa: Avatar (upload lên MinIO), Bio (tối đa 200 ký tự), Display Name (tối đa 50 ký tự)

### USER-03 Trạng thái hoạt động

Trạng thái: Online, Offline, Busy, Away, Studying, Coding
- Tự động chuyển Offline sau timeout
- Custom status text (tùy chọn)

### USER-04 Xem lịch sử hoạt động

Hiển thị: Phòng đã tham gia gần đây (30 ngày), Playlist gần đây, Bài hát đã nghe

---

## MODULE FRIEND (user-service)

| ID | Chức năng | Mô tả |
|----|-----------|-------|
| FRIEND-01 | Gửi lời mời kết bạn | Gửi friend request đến user khác |
| FRIEND-02 | Chấp nhận lời mời | Xác nhận kết bạn, tạo quan hệ hai chiều |
| FRIEND-03 | Từ chối lời mời | Từ chối friend request |
| FRIEND-04 | Hủy lời mời đã gửi | Cancel pending request |
| FRIEND-05 | Xóa bạn bè | Hủy quan hệ bạn bè |
| FRIEND-06 | Chặn người dùng | Block, ngăn mọi tương tác |
| FRIEND-07 | Bỏ chặn người dùng | Unblock user |

---

## MODULE ROOM (room-service)

### ROOM-01 Tạo phòng

**Input:** Tên phòng (bắt buộc), Mô tả (tùy chọn), Loại phòng (Public/Private/Friends-only), Mật khẩu (nếu Private)
**Giới hạn:** Mỗi user tạo tối đa 10 phòng

### ROOM-02 Chỉnh sửa phòng

Thay đổi: Tên, Ảnh bìa, Mô tả, Loại phòng, Mật khẩu

### ROOM-03 Xóa phòng

Chỉ Owner thực hiện, kick tất cả thành viên hiện tại

### ROOM-04 Tham gia phòng

Điều kiện: Room tồn tại và chưa đầy, có quyền truy cập (public/password/invite), không bị ban

### ROOM-05 Rời phòng

Nếu Owner rời → chuyển quyền cho Moderator có mặt lâu nhất, hoặc đóng phòng nếu không còn ai

### ROOM-06 Mời thành viên

Owner/Moderator gửi invite link hoặc mời trực tiếp

### ROOM-07 Kick thành viên

Moderator/Owner kick Member ra khỏi phòng (tạm thời)

### ROOM-08 Ban thành viên

Owner/Moderator ban Member (vĩnh viễn hoặc có thời hạn)

### ROOM-09 Mute thành viên

Moderator/Owner tắt quyền chat của Member trong phòng

### ROOM-10 Phân quyền

| Vai trò | Quyền hạn |
|---------|-----------|
| Owner | Toàn quyền: chỉnh sửa, xóa phòng, quản lý tất cả thành viên |
| Moderator | Kick, ban, mute Member; quản lý playlist |
| Member | Chat, nghe nhạc, vote, tham gia voice/video |

---

## MODULE PRESENCE (user-service / Redis)

| ID | Chức năng |
|----|-----------|
| PRESENCE-01 | Theo dõi người online trong hệ thống |
| PRESENCE-02 | Theo dõi danh sách người trong phòng |
| PRESENCE-03 | Theo dõi trạng thái cụ thể của user |
| PRESENCE-04 | Heartbeat session (ping mỗi 30s, timeout 90s) |

---

## MODULE CHAT (chat-service)

| ID | Chức năng | Chi tiết |
|----|-----------|----------|
| CHAT-01 | Gửi tin nhắn | Text, emoji, tối đa 2000 ký tự |
| CHAT-02 | Sửa tin nhắn | Trong vòng 15 phút sau khi gửi |
| CHAT-03 | Thu hồi tin nhắn | Xóa khỏi tất cả client (soft delete) |
| CHAT-04 | Xóa tin nhắn | Moderator/Owner xóa bất kỳ tin nhắn |
| CHAT-05 | Reply tin nhắn | Quote & reply một tin nhắn cụ thể |
| CHAT-06 | Mention người dùng | @username gửi notification cho người được mention |
| CHAT-07 | Reaction Emoji | Thêm/xóa emoji reaction trên tin nhắn |
| CHAT-08 | Pin Message | Ghim tin nhắn quan trọng (tối đa 10 pin/phòng) |
| CHAT-09 | Search Message | Full-text search trong lịch sử chat phòng |

---

## MODULE MUSIC SOURCE (music-service)

| ID | Chức năng | Input/Output |
|----|-----------|--------------|
| MUSIC-01 | YouTube | URL → Metadata (tên, artist, thumbnail, duration) |
| MUSIC-02 | Spotify | Spotify URL → Metadata |
| MUSIC-03 | Apple Music | Apple Music URL → Metadata |
| MUSIC-04 | SoundCloud | SoundCloud URL → Metadata |
| MUSIC-05 | Upload file | mp3/wav/flac, tối đa 50MB, lưu MinIO |
| MUSIC-06 | Lấy metadata | Tên, nghệ sĩ, thumbnail, duration, source URL |
| MUSIC-07 | Lưu lịch sử phát | Ghi nhận bài hát đã phát trong phòng |

---

## MODULE PLAYLIST (playlist-service)

| ID | Chức năng |
|----|-----------|
| PLAYLIST-01 | Tạo playlist (cá nhân hoặc phòng) |
| PLAYLIST-02 | Đổi tên playlist |
| PLAYLIST-03 | Xóa playlist |
| PLAYLIST-04 | Thêm bài hát vào playlist |
| PLAYLIST-05 | Xóa bài hát khỏi playlist |
| PLAYLIST-06 | Sắp xếp thứ tự bài hát (drag & drop) |
| PLAYLIST-07 | Upvote bài hát (tăng ưu tiên trong queue) |
| PLAYLIST-08 | Downvote bài hát |
| PLAYLIST-09 | Import playlist từ YouTube/Spotify |

---

## MODULE QUEUE (playback-service)

| ID | Chức năng |
|----|-----------|
| QUEUE-01 | Thêm bài vào queue |
| QUEUE-02 | Xóa bài khỏi queue |
| QUEUE-03 | Di chuyển vị trí bài trong queue |
| QUEUE-04 | Shuffle queue ngẫu nhiên |
| QUEUE-05 | Xóa toàn bộ queue |
| QUEUE-06 | Tự động phát bài tiếp theo |

---

## MODULE PLAYBACK (playback-service)

| ID | Chức năng | Mô tả |
|----|-----------|-------|
| PLAYBACK-01 | Play | Bắt đầu phát bài hát |
| PLAYBACK-02 | Pause | Tạm dừng |
| PLAYBACK-03 | Resume | Tiếp tục phát |
| PLAYBACK-04 | Stop | Dừng và reset position |
| PLAYBACK-05 | Seek | Di chuyển đến timestamp cụ thể |
| PLAYBACK-06 | Next Song | Phát bài tiếp theo trong queue |
| PLAYBACK-07 | Previous Song | Phát bài trước |
| PLAYBACK-08 | Đồng bộ thời gian phát | Broadcast timestamp đến tất cả client trong phòng (< 200ms) |
| PLAYBACK-09 | Đồng bộ trạng thái | Broadcast play/pause/seek state |
| PLAYBACK-10 | Khôi phục sau reconnect | Client reconnect → nhận lại state hiện tại từ Redis |

**Lưu ý:** Playback state lưu tại Redis với key `room:{id}:playback`, broadcast qua WebSocket.

---

## MODULE VOICE (voice-service / LiveKit)

| ID | Chức năng |
|----|-----------|
| VOICE-01 | Join Voice Channel (qua LiveKit Room) |
| VOICE-02 | Leave Voice Channel |
| VOICE-03 | Mute Microphone |
| VOICE-04 | Unmute Microphone |
| VOICE-05 | Chọn thiết bị âm thanh input/output |
| VOICE-06 | Điều chỉnh âm lượng từng người |
| VOICE-07 | Push To Talk (PTT mode) |

---

## MODULE VIDEO (voice-service / LiveKit)

| ID | Chức năng |
|----|-----------|
| VIDEO-01 | Bật camera |
| VIDEO-02 | Tắt camera |
| VIDEO-03 | Chọn camera device |
| VIDEO-04 | Grid Layout (hiển thị tất cả participants) |
| VIDEO-05 | Speaker Layout (focus người đang nói) |
| VIDEO-06 | Full Screen mode |

---

## MODULE SCREEN SHARE (voice-service / LiveKit)

| ID | Chức năng |
|----|-----------|
| SCREEN-01 | Share toàn màn hình |
| SCREEN-02 | Share cửa sổ ứng dụng cụ thể |
| SCREEN-03 | Share tab trình duyệt |
| SCREEN-04 | Dừng chia sẻ màn hình |

---

## MODULE NOTIFICATION (notification-service)

| ID | Loại thông báo |
|----|----------------|
| NOTI-01 | Ai đó tham gia phòng của bạn |
| NOTI-02 | Ai đó rời phòng |
| NOTI-03 | Bạn được @mention trong chat |
| NOTI-04 | Playlist/queue thay đổi |
| NOTI-05 | Có cuộc gọi đến |
| NOTI-06 | Friend request nhận được |
| NOTI-07 | Friend request được chấp nhận |

---

## MODULE SEARCH (search-service)

| ID | Chức năng | Mô tả |
|----|-----------|-------|
| SEARCH-01 | Tìm kiếm người dùng | Theo username, display name |
| SEARCH-02 | Tìm kiếm phòng | Theo tên, mô tả (chỉ public rooms) |
| SEARCH-03 | Tìm kiếm playlist | Theo tên playlist |
| SEARCH-04 | Tìm kiếm bài hát | Theo tên bài, nghệ sĩ |
| SEARCH-05 | Tìm kiếm tin nhắn | Full-text search trong phòng |

**Implementation:** PostgreSQL Full-Text Search (tsvector/tsquery), sau này có thể migrate sang Elasticsearch.

---

## MODULE ANALYTICS (analytics-service)

| ID | Chức năng |
|----|-----------|
| ANALYTICS-01 | Thống kê DAU/MAU |
| ANALYTICS-02 | Top phòng hoạt động (người dùng, thời gian) |
| ANALYTICS-03 | Top bài hát được phát nhiều nhất |
| ANALYTICS-04 | Thống kê thời gian sử dụng theo user/phòng |
| ANALYTICS-05 | Retention rate |
| ANALYTICS-06 | Thống kê lỗi và sự cố hệ thống |

---

## MODULE ADMIN (admin-service)

| ID | Chức năng |
|----|-----------|
| ADMIN-01 | Xem/tìm kiếm danh sách người dùng |
| ADMIN-02 | Khóa tài khoản (suspend) |
| ADMIN-03 | Mở khóa tài khoản |
| ADMIN-04 | Xóa phòng vi phạm |
| ADMIN-05 | Xem Audit Logs (mọi hành động admin) |
| ADMIN-06 | Dashboard hệ thống (metrics, active rooms, users online) |
| ADMIN-07 | Quản lý báo cáo vi phạm từ người dùng |

---

# 5. YÊU CẦU PHI CHỨC NĂNG

## 5.1 Hiệu năng (Performance)

### API Response Time

| Loại endpoint | P50 | P95 | P99 |
|---------------|-----|-----|-----|
| Auth (login/register) | < 100ms | < 300ms | < 500ms |
| User/Profile | < 50ms | < 150ms | < 300ms |
| Room CRUD | < 80ms | < 250ms | < 400ms |
| Chat (gửi tin nhắn) | < 50ms | < 100ms | < 200ms |
| Search | < 100ms | < 400ms | < 800ms |
| File upload (metadata) | < 200ms | < 500ms | < 1s |

### Realtime Events

| Loại event | P95 latency |
|-----------|-------------|
| Chat message broadcast | < 50ms |
| Playback sync | < 100ms |
| Presence update | < 200ms |
| Notification delivery | < 300ms |

### Database

- Query đơn giản (PK lookup): < 5ms
- Query phức tạp (join/filter): < 50ms
- Sử dụng connection pooling (pgx pgxpool): min 5, max 25 connections per service
- Tất cả queries phải có index phù hợp, không có full table scan ở production

### Throughput

- Hệ thống xử lý tối thiểu 1.000 requests/giây tổng hợp
- WebSocket: tối thiểu 10.000 concurrent connections
- Chat: tối thiểu 500 messages/giây

---

## 5.2 Khả năng mở rộng (Scalability)

### Horizontal Scaling

- Mỗi microservice phải **stateless**, scale horizontal bằng Kubernetes HPA
- Session/State lưu tại Redis, không lưu in-memory trong service
- WebSocket connections phải có **sticky session** hoặc dùng Redis Pub/Sub để broadcast cross-pod

### Capacity Targets

| Chỉ số | Mục tiêu v1.0 | Mục tiêu v2.0 |
|--------|---------------|---------------|
| Người dùng đăng ký | 50.000 | 500.000 |
| DAU (Daily Active Users) | 10.000 | 100.000 |
| Phòng hoạt động đồng thời | 1.000 | 10.000 |
| Concurrent WebSocket connections | 20.000 | 200.000 |
| Messages/ngày | 1.000.000 | 10.000.000 |

### Database Scaling

- **Read replicas:** 1 replica PostgreSQL cho read-heavy services (analytics, search)
- **Connection pooling:** PgBouncer trước PostgreSQL
- **Partitioning:** Bảng `messages` partition theo `room_id` và `created_at` (range partition theo tháng)
- **Archiving:** Messages > 1 năm chuyển sang cold storage

---

## 5.3 Độ khả dụng (Availability)

| Chỉ số | Mục tiêu |
|--------|----------|
| Uptime tổng hệ thống | ≥ 99.5% (≤ 43.8h downtime/năm) |
| Core services (auth, room, chat) | ≥ 99.9% |
| Planned maintenance window | < 2h/tháng, thông báo trước 48h |

### Fault Tolerance

- **Circuit Breaker** cho tất cả service-to-service calls (dùng go-circuitbreaker hoặc hystrix-go)
- **Retry with exponential backoff** cho transient failures
- **Graceful degradation:** Nếu analytics-service down → hệ thống chính tiếp tục hoạt động bình thường
- **Health checks:** `/health` và `/ready` endpoint trên mỗi service (Kubernetes liveness/readiness probe)
- **Kubernetes:** Minimum 2 replicas cho tất cả core services (auto-restart on failure)

### Backup

- PostgreSQL: Daily full backup + WAL streaming replication
- Redis: AOF persistence + daily snapshot
- MinIO: Replication sang ít nhất 2 nodes
- Retention: 30 ngày cho daily backup, 7 ngày cho WAL

---

## 5.4 Bảo mật (Security)

### Authentication & Authorization

- **JWT:** Access Token signed bằng RS256 (asymmetric), TTL 15 phút
- **Refresh Token:** Stored as httpOnly cookie hoặc lưu server-side với rotation
- **OAuth2:** Google OAuth2, state parameter chống CSRF
- **RBAC:** Role-based access control cho từng resource (Owner/Moderator/Member)
- **Row-level security:** PostgreSQL RLS cho multi-tenant data isolation giữa các phòng

### Network Security

- **HTTPS/TLS 1.3** bắt buộc cho tất cả endpoints
- **WebSocket:** WSS (WebSocket Secure) only
- **CORS:** Whitelist origins cụ thể, không dùng `*`
- **Security Headers:** HSTS, X-Frame-Options, CSP, X-Content-Type-Options

### Rate Limiting

| Endpoint | Limit |
|----------|-------|
| POST /auth/login | 5 requests/phút/IP |
| POST /auth/register | 3 requests/phút/IP |
| POST /auth/forgot-password | 3 requests/giờ/email |
| GET /search/* | 30 requests/phút/user |
| POST /chat/messages | 60 messages/phút/user |
| General API | 100 requests/phút/user |

- Rate limiting lưu tại Redis (sliding window algorithm)
- Trả về HTTP 429 với `Retry-After` header khi vượt giới hạn

### Data Security

- **Password hashing:** bcrypt với cost factor ≥ 12
- **Sensitive data encryption:** Mã hóa PII (email, phone) tại rest nếu cần
- **SQL Injection:** Dùng parameterized queries qua sqlc/pgx (không raw SQL interpolation)
- **Input validation & sanitization:** Validate tất cả input tại API Gateway và service layer
- **File upload security:** Validate MIME type, scan virus (ClamAV), giới hạn kích thước 50MB
- **Secret management:** Kubernetes Secrets hoặc HashiCorp Vault cho credentials/API keys

### Audit & Compliance

- **Audit Logging:** Ghi lại tất cả hành động admin, login/logout, thay đổi phòng
- **Log format:** Structured JSON logs với `user_id`, `action`, `resource`, `ip`, `timestamp`
- **Data retention:** Logs giữ tối thiểu 90 ngày
- **GDPR-ready:** Có endpoint `DELETE /users/me` để xóa hoàn toàn dữ liệu người dùng

---

## 5.5 Giám sát và Vận hành (Observability)

### Metrics (Prometheus)

Mỗi service expose `/metrics` endpoint với:
- HTTP request rate, latency (histogram), error rate (RED method)
- WebSocket active connections
- Database connection pool usage
- Redis hit/miss rate
- Queue depth (message queue)
- Custom business metrics: active rooms, online users, messages/min

### Logging (Loki + Grafana)

- Log level: DEBUG (dev), INFO (prod), ERROR (always)
- Structured JSON format với `trace_id` cho distributed tracing
- Log aggregation: Loki → hiển thị trên Grafana
- **Không log:** Password, token, PII data

### Tracing (Jaeger / OpenTelemetry)

- Distributed tracing qua tất cả microservices
- Trace ID propagate qua HTTP header `X-Trace-ID`
- Sample rate: 10% ở production, 100% ở staging

### Alerting (Alertmanager)

| Alert | Ngưỡng | Severity |
|-------|--------|----------|
| High error rate | Error rate > 1% trong 5 phút | Critical |
| High latency | P95 > 500ms trong 10 phút | Warning |
| Service down | Health check fail > 1 phút | Critical |
| DB connection exhausted | Pool usage > 90% | Warning |
| Disk usage | > 80% | Warning |

### Dashboard Grafana

- **System Overview:** CPU, Memory, Network, Disk trên tất cả nodes
- **Service Health:** Request rate, latency, error rate theo service
- **Business Metrics:** Active users, active rooms, messages/s
- **Database:** Query performance, connections, replication lag

---

## 5.6 Khả năng Bảo trì (Maintainability)

### Code Quality

- **Test coverage:** Tối thiểu 70% unit test coverage cho business logic
- **Integration tests:** Tất cả API endpoints phải có integration test
- **Code review:** Bắt buộc PR review trước khi merge vào main
- **Linting:** golangci-lint với ruleset chuẩn

### Deployment

- **Zero-downtime deployment:** Rolling update trên Kubernetes
- **Feature flags:** Hỗ trợ tắt/bật feature mà không cần deploy lại
- **Database migration:** Dùng golang-migrate, migration phải backward-compatible
- **Rollback:** Có thể rollback service về version trước trong < 5 phút

### Documentation

- **API Documentation:** Swagger/OpenAPI 3.0 tự động generate từ code
- **Architecture Decision Records (ADR):** Ghi lại các quyết định kiến trúc quan trọng
- **Runbook:** Hướng dẫn xử lý sự cố cho từng loại alert

---

## 5.7 Tương thích (Compatibility)

### Browser Support

| Browser | Phiên bản tối thiểu |
|---------|---------------------|
| Chrome | 90+ |
| Firefox | 88+ |
| Safari | 14+ |
| Edge | 90+ |

### Devices

- Desktop (Windows, macOS, Linux)
- Mobile Web (iOS Safari 14+, Android Chrome 90+)
- Responsive design: Hỗ trợ màn hình từ 360px đến 2560px

### API Versioning

- URL versioning: `/api/v1/...`
- Khi breaking change → tạo `/api/v2/...`, duy trì v1 tối thiểu 6 tháng
- Deprecation notice qua response header: `Deprecation: true`, `Sunset: <date>`

---

# 6. YÊU CẦU GIAO DIỆN NGƯỜI DÙNG (UI/UX)

## 6.1 Theme & Appearance

### Dark Mode / Light Mode

Hệ thống **bắt buộc hỗ trợ cả Dark Mode và Light Mode** với các yêu cầu sau:

**Palette màu sắc tối thiểu:**

| Token | Dark Mode | Light Mode | Mô tả |
|-------|-----------|------------|-------|
| `--bg-primary` | `#0f1117` | `#ffffff` | Nền chính |
| `--bg-secondary` | `#1a1d27` | `#f5f5f5` | Nền phụ (sidebar, panel) |
| `--bg-surface` | `#22263a` | `#e8e8e8` | Card, modal, dropdown |
| `--bg-hover` | `#2a2f45` | `#d8d8d8` | Hover state |
| `--text-primary` | `#e8eaf0` | `#1a1a2e` | Text chính |
| `--text-secondary` | `#8b92a5` | `#555770` | Text phụ, placeholder |
| `--text-muted` | `#555c70` | `#999aaa` | Text mờ, disabled |
| `--accent-primary` | `#7c6ff7` | `#5a52d0` | Brand color chính |
| `--accent-hover` | `#9088f9` | `#4840c0` | Hover accent |
| `--border` | `#2e3347` | `#d0d0d0` | Đường viền |
| `--danger` | `#e85d75` | `#d63050` | Lỗi, xóa, nguy hiểm |
| `--success` | `#43c67e` | `#1e9e58` | Thành công, online |
| `--warning` | `#f5a623` | `#d4870e` | Cảnh báo |

**Yêu cầu kỹ thuật:**
- Dùng **CSS Custom Properties (CSS Variables)** để quản lý theme
- Phát hiện và áp dụng **system preference** (`prefers-color-scheme`) khi lần đầu truy cập
- Người dùng có thể **override** và lưu preference vào `localStorage` + backend (user settings)
- **Transition mượt** khi chuyển theme: `transition: background 0.2s, color 0.2s`
- Không để text #fff trên nền trắng hay ngược lại (contrast ratio ≥ 4.5:1 theo WCAG AA)

---

## 6.2 Layout & Navigation

### Desktop Layout (≥ 1024px)

```
┌────────────────────────────────────────────────────────┐
│  Top Bar: Logo | Search | User Avatar | Notifications  │
├───────┬────────────────────────────┬───────────────────┤
│       │                            │                   │
│ Left  │     Main Content Area      │   Right Panel     │
│ Sidebar│   (Room / Chat / Player) │   (Members list,  │
│(Nav)  │                            │    Queue, etc.)   │
│       │                            │                   │
├───────┴────────────────────────────┴───────────────────┤
│  Music Player Bar (persistent bottom bar)              │
└────────────────────────────────────────────────────────┘
```

**Left Sidebar:** Danh sách phòng đã join, friends list, navigation icons, user avatar + status
**Main Content:** Khu vực chính (room view, chat, video grid)
**Right Panel:** Collapsible, hiển thị danh sách thành viên, queue nhạc
**Player Bar:** Fixed bottom, luôn hiển thị khi đang phát nhạc

### Mobile Layout (< 768px)

- **Bottom Navigation Bar** thay thế Left Sidebar (Home, Rooms, Search, Profile)
- Right Panel ẩn, mở qua bottom sheet khi cần
- Player Bar thu nhỏ, tap để expand
- Voice/Video chiếm toàn màn hình khi active

### Tablet Layout (768px – 1023px)

- Left Sidebar thu nhỏ (icon only), hover/tap để expand
- Right Panel ẩn mặc định, có nút toggle

---

## 6.3 Typography

| Loại | Font | Size | Weight |
|------|------|------|--------|
| Heading 1 | Inter / System UI | 28px | 700 |
| Heading 2 | Inter / System UI | 22px | 600 |
| Heading 3 | Inter / System UI | 18px | 600 |
| Body | Inter / System UI | 14px | 400 |
| Small / Caption | Inter / System UI | 12px | 400 |
| Code / Mono | JetBrains Mono / monospace | 13px | 400 |
| Chat messages | Inter / System UI | 14px | 400 |

- **Font stack:** `"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`
- Line height body: 1.5
- Không dùng font size < 12px

---

## 6.4 Component Library (Tối thiểu)

Các component phải được xây dựng theo **Design System** thống nhất, hỗ trợ đầy đủ dark/light mode:

### Buttons

| Variant | Mô tả |
|---------|-------|
| Primary | Accent color, hành động chính |
| Secondary | Ghost/outline, hành động phụ |
| Danger | Màu đỏ, xóa/hủy |
| Icon Button | Chỉ icon, có tooltip |
| Loading State | Spinner animation khi đang xử lý |

### Form Elements

- Input field: Border + focus ring với accent color
- Password input: Toggle show/hide
- Textarea: Auto-resize
- Checkbox, Radio, Toggle Switch
- Dropdown/Select
- File upload drop zone

### Feedback Components

- Toast/Snackbar: Góc dưới phải, tự dismiss sau 3-5s, có variants success/error/warning/info
- Modal/Dialog: Backdrop blur, Escape key để đóng, focus trap
- Tooltip: Hover delay 200ms
- Skeleton loader: Khi đang fetch data (thay vì spinner cho cả trang)
- Empty state: Illustration + message khi không có dữ liệu

### Chat Components

- Message bubble: Avatar, username, timestamp, hover actions (reply, react, more)
- Emoji picker: Grid layout, search, recent emojis
- @mention autocomplete dropdown
- Link preview card
- File/image attachment preview

### Room Components

- Member list item: Avatar + status indicator (màu dot: xanh=online, vàng=away, đỏ=busy)
- Queue item: Thumbnail + title + artist + duration + vote buttons
- Now playing: Animated waveform hoặc spinning disc
- Playback controls: Play/Pause, Prev, Next, Seek bar, Volume, duration display

---

## 6.5 Accessibility (A11y)

- **WCAG 2.1 Level AA** tối thiểu
- Tất cả interactive elements có `:focus-visible` style rõ ràng
- Icon-only buttons phải có `aria-label`
- Contrast ratio text/background ≥ 4.5:1 (cả dark và light mode)
- Keyboard navigation đầy đủ (Tab, Shift+Tab, Enter, Space, Arrow keys)
- Screen reader support: Semantic HTML, ARIA roles khi cần
- Không phụ thuộc màu sắc đơn thuần để truyền thông tin (kèm icon hoặc text)

---

## 6.6 Loading & Error States

### Loading States

- **Page load:** Skeleton screens thay vì spinner toàn trang
- **Data fetch:** Inline skeleton cho từng component
- **Action buttons:** Disabled + spinner khi đang submit
- **Infinite scroll / Pagination:** Loading indicator ở cuối list

### Error States

- **Network error:** Toast thông báo + retry button
- **Form validation:** Inline error message ngay dưới field bị lỗi
- **404 Not Found:** Trang thông báo rõ ràng + link về trang chủ
- **503 Service Unavailable:** Maintenance page với thông báo dự kiến thời gian phục hồi
- **Empty Search Results:** Minh họa + gợi ý tìm kiếm khác

---

## 6.7 Animation & Transitions

- **Theme switch:** 200ms ease transition
- **Modal open/close:** Fade + scale (150ms)
- **Sidebar expand/collapse:** 200ms ease
- **Toast appear:** Slide-in từ dưới (200ms)
- **Message appear:** Fade-in mới (100ms)
- **Page transition:** Fade (100ms)
- **Tránh animation làm chậm interaction** (không dùng transition > 300ms cho thao tác người dùng)
- **Respect `prefers-reduced-motion`:** Tắt animation nếu user cài reduced motion

---

## 6.8 Internationalization (i18n) — Phạm vi tương lai

- Thiết kế cấu trúc support i18n ngay từ đầu
- **MVP:** Tiếng Việt và Tiếng Anh
- Sử dụng translation keys, không hardcode string trong component
- Format ngày/giờ theo locale (dùng `Intl.DateTimeFormat`)
- RTL support: Để dành, không bắt buộc ở v1.0

---

# 7. PHIÊN BẢN MVP

## Phạm vi MVP (Priority 1)

### Backend Services

| Service | Scope MVP |
|---------|-----------|
| auth-service | Đăng ký, đăng nhập, Google OAuth, JWT |
| user-service | Profile, presence (online/offline) |
| room-service | CRUD phòng, join/leave, phân quyền cơ bản |
| chat-service | Gửi/nhận tin nhắn, WebSocket |
| music-service | YouTube source, metadata |
| playback-service | Play/Pause/Seek, đồng bộ state, queue |
| voice-service | Join/leave voice, mute/unmute (LiveKit) |
| notification-service | In-app notification cơ bản |

### Infrastructure MVP

| Thành phần | Yêu cầu |
|------------|---------|
| PostgreSQL | Single instance + 1 replica |
| Redis | Single instance |
| MinIO | Single node |
| WebSocket | chat-service + playback-service |
| Docker Compose | Dev/Staging environment |
| Kubernetes | Production (basic, không autoscaling) |
| GitHub Actions | Build, test, deploy pipeline |

### UI MVP

- Dark Mode và Light Mode (bắt buộc)
- Responsive: Desktop + Mobile
- Đăng ký/đăng nhập/profile
- Room: tạo, tham gia, chat
- Music player: play/pause/seek, queue
- Voice call cơ bản
- Toast notifications

## Không bao gồm trong MVP (Priority 2 – v1.1+)

- Video Call
- Screen Sharing
- Spotify/Apple Music/SoundCloud integration
- Advanced analytics
- Full admin dashboard
- Import playlist
- Vote/Downvote system
- Advanced search
- Push notifications (mobile)
- GDPR data export/delete

---

*Tài liệu này sẽ được cập nhật liên tục trong quá trình phát triển. Mọi thay đổi cần được review bởi Tech Lead trước khi merge.*

**Version History:**

| Phiên bản | Ngày | Thay đổi |
|-----------|------|---------|
| 1.0 | 07/06/2026 | Phiên bản ban đầu |
| 1.1 | 07/06/2026 | Bổ sung kiến trúc microservices, NFR chi tiết, UI/UX requirements |