# WorkTogether (SyncSpace)

Bilingual documentation: [English Version](#english-version) | [Phiên bản Tiếng Việt](#phiên-bản-tiếng-việt)

---

## English Version

WorkTogether (SyncSpace) is a low-latency, real-time collaboration workspace platform built with **Go Microservices** and **Angular**. It allows multiple users to join online spaces to stream music synchronously, chat, share screens, make voice/video calls, and collaborate in real-time.

### Core Features

- **Authentication (auth-service)**: Google OAuth2, secure JWT authentication with token rotation, and email verification.
- **User Management (user-service)**: Profile customisation, friend requests, user blocks, and activity logs.
- **Rooms & Permissions (room-service)**: Space administration, moderator/member/owner roles, and invite links.
- **Presence & Vibe (user-service / Redis)**: Heartbeat-based online presence tracking, custom user status, and room vibe metrics.
- **Chat (chat-service)**: Real-time messaging over WebSockets, emoji reactions, pinned messages, message history, and full-text search.
- **Synced Playback (playback-service)**: Room audio player synchronized across all users (< 200ms latency), state recovery on reconnect.
- **Music Catalog (music-service)**: YouTube/Spotify stream metadata fetching, file upload (MinIO storage), and playback history tracking.
- **Playlists & Queues (playlist-service & playback-service)**: Collaborative playlist CRUD operations, song queues, and priority voting.
- **Voice, Video & Screen Share (voice-service / LiveKit)**: High-quality, low-latency audio/video conference rooms and screen sharing.
- **Real-Time Notifications (notification-service)**: Dynamic notification delivery for calls, mentions, and friend requests.
- **Collaborative Editor (collab-service)**: Live, real-time code/document shared editing sessions.

---

### System Architecture

```text
                  ┌───────────────────────────────────┐
                  │          Web Client / UI          │
                  │       (Angular / Vanilla JS)      │
                  └─────────────────┬─────────────────┘
                                    │ HTTP / WSS (WebSockets)
                                    ▼
                  ┌───────────────────────────────────┐
                  │         NGINX API Gateway         │
                  │            (Port 8080)            │
                  └──────┬────────────────────┬───────┘
                         │                    │
        ┌────────────────┘                    └────────────────┐
        │ REST / gRPC                                          │ REST / gRPC
        ▼                                                      ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ Auth Service  │  │ Room Service  │  │ Chat Service  │  │ Playback Svc  │  │ Music Service │
│  (Port 8081)  │  │  (Port 8083)  │  │  (Port 8084)  │  │  (Port 8087)  │  │  (Port 8085)  │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │                  │                  │                  │
┌───────┴───────┐  ┌───────┴───────┐  ┌───────┴───────┐  ┌───────┴───────┐  ┌───────┴───────┐
│ User Service  │  │ Voice Service │  │ Playlist Svc  │  │ Notif Service │  │ Collab Svc    │
│  (Port 8082)  │  │  (Port 8088)  │  │  (Port 8086)  │  │  (Port 8089)  │  │  (Port 8094)  │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │
┌───────┴───────┐          │
│ Timer Service │          │ LiveKit SDK
│  (Port 8093)  │          ▼
└───────────────┘  ┌───────────────┐
                   │ LiveKit Server│
                   │  (Port 7880)  │
                   └───────────────┘
                           │
                           │ (Shared Databases & Infra)
                           ▼
        ┌──────────────────┼──────────────────┬──────────────────┐
        │                  │                  │                  │
        ▼                  ▼                  ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  PostgreSQL  │   │    Redis     │   │    MinIO     │   │ Redis Streams│
│ (Port 5433)  │   │ (Port 6379)  │   │ (Port 9000)  │   │ (Message Bus)│
└──────────────┘   └──────────────┘   └──────────────┘   └──────────────┘
```

---

### Project Directory Structure

```text
├── services/               # Go microservices
│   ├── auth-service/       # Authentication service
│   ├── chat-service/       # Message & WebSocket hub
│   ├── collab-service/     # Collaborative editing service
│   ├── music-service/      # Track metadata & storage
│   ├── notification-service/
│   ├── playback-service/   # Sync state manager
│   ├── playlist-service/
│   ├── room-service/       # Room configurations & roles
│   ├── timer-service/      # Pomodoro/room timers
│   ├── user-service/       # Profiles & friends
│   └── voice-service/      # LiveKit interface
├── web-client/             # Angular client app
├── web-client-vanilla/     # Vanilla HTML/JS client (for testing)
├── docker/                 # Service-specific docker configurations (Nginx, LiveKit, DB init)
├── docs/                   # API Specifications and Roadmaps
└── srs.md                  # Software Requirements Specification (Vietnamese)
```

---

### Getting Started

#### Prerequisites

Make sure you have the following installed on your machine:
- [Docker & Docker Compose](https://www.docker.com/)
- [Go (1.21+)](https://go.dev/) (optional, if running services locally outside Docker)
- [Node.js (v18+) & npm](https://nodejs.org/) (for running the frontend)

#### Step 1: Environment Setup

1. Copy `.env.example` to `.env` in the root directory:
   ```bash
   cp .env.example .env
   ```
2. Open `.env` and fill in the required credentials and configuration values (e.g., SMTP details, Google OAuth client secrets, and strong DB passwords).

#### Step 2: Start Infrastructure & Services

Launch all database, caching, LiveKit infrastructure, and microservices in detached mode:
```bash
docker-compose up -d
```
Verify that all containers are running successfully:
```bash
docker-compose ps
```

#### Step 3: Run Database Migrations

WorkTogether databases need to have schemas populated. Use the provided migration runner.

- **On Windows (PowerShell)**:
  ```powershell
  .\migrate.ps1 up
  ```
- **On Linux / macOS**:
  Ensure you have PowerShell (`pwsh`) installed, or run the migration script with:
  ```bash
  pwsh ./migrate.ps1 up
  ```

#### Step 4: Run the Client Application

##### 1. Angular Client (Recommended)
1. Navigate to the client directory:
   ```bash
   cd web-client
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Run the development server:
   ```bash
   npm run start
   # or
   ng serve
   ```
4. Access the UI at `http://localhost:4200/`.

##### 2. Vanilla JS Client (Testing/Minimalist)
1. Navigate to the vanilla client directory:
   ```bash
   cd web-client-vanilla
   ```
2. Run using a local dev server, or install dependencies and start:
   ```bash
   npm install
   npm run dev
   ```
3. Access the minimalist UI at the dev server port shown in your terminal.

---

### Ports Reference Table

| Service / Component | Internal Port | Host Port | Database / Schema / Key-space |
| :--- | :--- | :--- | :--- |
| **API Gateway (Nginx)** | 8080 | **8080** | N/A |
| **auth-service** | 8081 | N/A | `worktogether_auth` |
| **user-service** | 8082 | N/A | `worktogether_user` |
| **room-service** | 8083 | N/A | `worktogether_room` |
| **chat-service** | 8084 | N/A | `worktogether_chat` |
| **music-service** | 8085 | N/A | `worktogether_music` |
| **playlist-service** | 8086 | N/A | `worktogether_playlist` |
| **playback-service** | 8087 | N/A | Redis caching / Sync state |
| **voice-service** | 8088 | N/A | LiveKit interface |
| **notification-service**| 8089 | N/A | `worktogether_notification` |
| **timer-service** | 8093 | N/A | Redis caching |
| **collab-service** | 8094 | N/A | `worktogether_collab` |
| **PostgreSQL** | 5432 | **5433** | Shared DB instance |
| **Redis** | 6379 | **6379** | Shared Cache & Pub/Sub |
| **MinIO Console** | 9001 | **9001** | Music & Avatar uploads |
| **LiveKit Server** | 7880 | **7880** | Voice/Video room engine |

---

## Phiên bản Tiếng Việt

WorkTogether (SyncSpace) là nền tảng không gian làm việc và học tập cộng tác thời gian thực với độ trễ thấp, được xây dựng dựa trên kiến trúc **Go Microservices** và **Angular**. Hệ thống cho phép nhiều người dùng tham gia cùng một không gian trực tuyến để nghe nhạc đồng bộ, chat, chia sẻ màn hình và gọi voice/video call.

### Các Tính Năng Chính

- **Xác thực (auth-service)**: Đăng nhập Google OAuth2, xác thực JWT bảo mật với cơ chế xoay vòng Refresh Token và xác thực email.
- **Quản lý người dùng (user-service)**: Tùy chỉnh trang cá nhân, gửi yêu cầu kết bạn, chặn người dùng và lưu lịch sử hoạt động.
- **Quản lý phòng (room-service)**: Phân quyền quản trị không gian học tập (Room Owner, Moderator, Member) và lời mời tham gia phòng.
- **Trạng thái hoạt động & Vibe (user-service / Redis)**: Theo dõi trạng thái online thời gian thực bằng cơ chế Heartbeat, đặt status cá nhân, đo lường điểm tương tác (vibe) của phòng.
- **Trò chuyện (chat-service)**: Nhắn tin trực tuyến qua WebSocket, reaction emoji, ghim tin nhắn quan trọng, tìm kiếm lịch sử chat.
- **Đồng bộ phát nhạc (playback-service)**: Trình phát nhạc đồng bộ giữa tất cả thành viên trong phòng (độ trễ < 200ms), tự động khôi phục trạng thái phát khi kết nối lại.
- **Kho âm nhạc (music-service)**: Trích xuất thông tin bài hát từ YouTube/Spotify, cho phép tải lên file nhạc trực tiếp (lưu trữ MinIO) và lưu lịch sử phát nhạc.
- **Danh sách phát (playlist-service & playback-service)**: Quản lý playlist cộng tác, hàng đợi nhạc (queue), upvote/downvote bài hát để ưu tiên phát.
- **Voice, Video & Share màn hình (voice-service / LiveKit)**: Họp thoại/họp video chất lượng cao qua LiveKit Server, chia sẻ màn hình, tùy chỉnh thiết bị âm thanh.
- **Thông báo thời gian thực (notification-service)**: Hệ thống thông báo in-app và push-notification cho cuộc gọi, lượt nhắc tên (@mention), và lời mời kết bạn.
- **Trình soạn thảo cộng tác (collab-service)**: Soạn thảo mã nguồn/tài liệu thời gian thực chung giữa các thành viên.

---

### Kiến Trúc Hệ Thống

Xem [Sơ đồ Kiến trúc dạng ASCII ở phần tiếng Anh](#system-architecture). Luồng đi chính của ứng dụng: Web Client -> Nginx API Gateway (Port 8080) -> Hệ thống Microservices Go -> Các Cơ sở dữ liệu và hạ tầng dùng chung (PostgreSQL, Redis, MinIO, LiveKit).

---

### Cấu Trúc Thư Mục Dự Án

```text
├── services/               # Các microservices viết bằng Go
│   ├── auth-service/       # Service xác thực
│   ├── chat-service/       # Quản lý tin nhắn & WebSocket hub
│   ├── collab-service/     # Soạn thảo code/tài liệu cộng tác
│   ├── music-service/      # Quản lý thông tin bài hát & file upload
│   ├── notification-service/
│   ├── playback-service/   # Đồng bộ trạng thái phát nhạc
│   ├── playlist-service/
│   ├── room-service/       # Quản lý phòng & phân quyền
│   ├── timer-service/      # Bộ đếm giờ (Pomodoro)
│   ├── user-service/       # Trang cá nhân & bạn bè
│   └── voice-service/      # Kết nối LiveKit
├── web-client/             # Ứng dụng client viết bằng Angular
├── web-client-vanilla/     # Ứng dụng client tối giản (HTML/JS thuần để test)
├── docker/                 # File cấu hình docker cho từng thành phần (Nginx, LiveKit, v.v.)
├── docs/                   # Tài liệu mô tả API & Lộ trình phát triển
└── srs.md                  # Tài liệu đặc tả yêu cầu phần mềm (SRS)
```

---

### Hướng Dẫn Cài Đặt & Khởi Chạy

#### Yêu cầu hệ thống

Trước khi bắt đầu, hãy đảm bảo máy tính của bạn đã được cài đặt:
- [Docker & Docker Compose](https://www.docker.com/)
- [Go (1.21+)](https://go.dev/) (không bắt buộc, chỉ dùng nếu chạy trực tiếp dịch vụ không qua Docker)
- [Node.js (v18+) & npm](https://nodejs.org/) (để chạy ứng dụng Web Client)

#### Bước 1: Thiết lập cấu hình môi trường

1. Tạo file cấu hình `.env` từ file ví dụ `.env.example`:
   ```bash
   cp .env.example .env
   ```
2. Mở file `.env` vừa tạo và điền các thông số cần thiết (ví dụ: cấu hình gửi mail SMTP, mã khóa bí mật cho Google OAuth2 Client, mật khẩu cơ sở dữ liệu).

#### Bước 2: Khởi chạy Hạ tầng và Services

Chạy toàn bộ cơ sở dữ liệu, bộ nhớ đệm, LiveKit Server và các microservices ở chế độ nền (detached mode):
```bash
docker-compose up -d
```
Kiểm tra danh sách và trạng thái của các container:
```bash
docker-compose ps
```

#### Bước 3: Chạy Database Migrations

Để tạo các bảng và schema ban đầu cho hệ thống cơ sở dữ liệu:

- **Trên Windows (PowerShell)**:
  ```powershell
  .\migrate.ps1 up
  ```
- **Trên Linux / macOS**:
  Đảm bảo bạn đã cài đặt PowerShell (`pwsh`), sau đó chạy script bằng lệnh:
  ```bash
  pwsh ./migrate.ps1 up
  ```

#### Bước 4: Khởi chạy Client (Giao diện)

##### 1. Angular Client (Giao diện chính thức)
1. Di chuyển vào thư mục client:
   ```bash
   cd web-client
   ```
2. Cài đặt các thư viện phụ thuộc:
   ```bash
   npm install
   ```
3. Khởi chạy development server:
   ```bash
   npm run start
   # hoặc
   ng serve
   ```
4. Truy cập giao diện tại địa chỉ `http://localhost:4200/`.

##### 2. Vanilla JS Client (Bản tối giản dùng để thử nghiệm nhanh)
1. Di chuyển vào thư mục vanilla client:
   ```bash
   cd web-client-vanilla
   ```
2. Cài đặt và khởi động:
   ```bash
   npm install
   npm run dev
   ```
3. Truy cập link local dev server hiển thị trên màn hình terminal của bạn.

---

### Danh Sách Cổng Dịch Vụ (Port Reference)

Xem [Bảng ánh xạ cổng dịch vụ ở phần tiếng Anh](#ports-reference-table) để biết thông tin chi tiết về cổng chạy của từng Microservice, Database (PostgreSQL port 5433, Redis port 6379, MinIO Console port 9001) và LiveKit (port 7880).
