# Phase 2: Nghe Cùng Nhau Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Triển khai toàn bộ tính năng Phase 2 (Synced Lyrics, Bookmarks, Realtime Poll, Guest DJ) cho cả Backend (Go) và Frontend (Angular).

**Architecture:** Bổ sung cơ sở dữ liệu và REST API tương ứng trong `music-service` cho Lời bài hát & Bookmark. Mở rộng WebSocket Hub và Redis state trong `playback-service` cho cơ chế Guest DJ và Realtime Poll. Cập nhật giao diện Web Client (Angular) tương ứng.

**Tech Stack:** Go (Gin, sqlx, go-redis), Angular 19 (TypeScript, RxJS).

## Global Constraints
- Viết mã nguồn sạch, tôn trọng nguyên tắc Onion/Clean Architecture trong các microservices Go.
- Thiết kế UI/UX theo Cool Ocean design system, kế thừa các CSS custom properties từ phòng.
- Mọi API mới phải đi kèm tài liệu và unit test đầy đủ.

---

### Task 1: Database Migration for Lyrics & Bookmarks (`music-service`)

**Files:**
- Create: `services/music-service/db/migrations/000002_add_lyrics_and_bookmarks.up.sql`
- Modify: `services/music-service/internal/repository/postgres.go`

**Interfaces:**
- Consumes: PostgreSQL DB connection instance in repository package.
- Produces: `track_lyrics` và `bookmarks` tables in Postgres database.

- [ ] **Step 1: Write migration SQL file**
  Create file `services/music-service/db/migrations/000002_add_lyrics_and_bookmarks.up.sql` with:
  ```sql
  CREATE TABLE IF NOT EXISTS track_lyrics (
      track_id VARCHAR(36) PRIMARY KEY,
      content TEXT NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      CONSTRAINT fk_track FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
  );

  CREATE TABLE IF NOT EXISTS bookmarks (
      id VARCHAR(36) PRIMARY KEY,
      room_id VARCHAR(36) NOT NULL,
      user_id VARCHAR(36) NOT NULL,
      track_id VARCHAR(36) NOT NULL,
      position_ms INTEGER NOT NULL,
      note VARCHAR(255) NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      CONSTRAINT fk_track_bookmark FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE
  );
  CREATE INDEX IF NOT EXISTS idx_bookmarks_room ON bookmarks(room_id);
  ```

- [ ] **Step 2: Add table initialization logic in postgres.go**
  Modify `initTables` method in `services/music-service/internal/repository/postgres.go` around line 55 to run the schema definitions:
  ```go
  // Inside initTables:
  lyricsSchema := `
  CREATE TABLE IF NOT EXISTS track_lyrics (
      track_id VARCHAR(36) PRIMARY KEY REFERENCES tracks(id) ON DELETE CASCADE,
      content TEXT NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
  );`
  if _, err := r.db.ExecContext(ctx, lyricsSchema); err != nil {
      return err
  }

  bookmarksSchema := `
  CREATE TABLE IF NOT EXISTS bookmarks (
      id VARCHAR(36) PRIMARY KEY,
      room_id VARCHAR(36) NOT NULL,
      user_id VARCHAR(36) NOT NULL,
      track_id VARCHAR(36) REFERENCES tracks(id) ON DELETE CASCADE,
      position_ms INTEGER NOT NULL,
      note VARCHAR(255) NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
  );`
  if _, err := r.db.ExecContext(ctx, bookmarksSchema); err != nil {
      return err
  }
  ```

- [ ] **Step 3: Run migration verification**
  Run: `docker compose restart music-service`
  Expected: Service starts cleanly, prints logs: "Đã khởi tạo schema PostgreSQL cho music-service thành công."

- [ ] **Step 4: Commit**
  ```bash
  git add services/music-service/db/migrations/000002_add_lyrics_and_bookmarks.up.sql services/music-service/internal/repository/postgres.go
  git commit -m "migration: add track_lyrics and bookmarks tables"
  ```

---

### Task 2: Model & Repository implementation for Lyrics (`music-service`)

**Files:**
- Modify: `services/music-service/internal/domain/models.go`
- Modify: `services/music-service/internal/repository/postgres.go`
- Create: `services/music-service/internal/repository/postgres_test.go`

**Interfaces:**
- Consumes: PostgresRepository DB connection
- Produces:
  ```go
  SaveLyrics(ctx context.Context, trackID string, content string) error
  GetLyrics(ctx context.Context, trackID string) (string, error)
  ```

- [ ] **Step 1: Update models.go**
  Add structures in `services/music-service/internal/domain/models.go`:
  ```go
  type TrackLyrics struct {
      TrackID   string    `json:"track_id"`
      Content   string    `json:"content"`
      UpdatedAt time.Time `json:"updated_at"`
  }
  ```

- [ ] **Step 2: Add database methods in postgres.go**
  Implement the following functions at the end of `services/music-service/internal/repository/postgres.go`:
  ```go
  func (r *PostgresRepository) SaveLyrics(ctx context.Context, trackID string, content string) error {
      query := `
      INSERT INTO track_lyrics (track_id, content, updated_at)
      VALUES ($1, $2, NOW())
      ON CONFLICT (track_id) DO UPDATE SET content = EXCLUDED.content, updated_at = NOW()`
      _, err := r.db.ExecContext(ctx, query, trackID, content)
      return err
  }

  func (r *PostgresRepository) GetLyrics(ctx context.Context, trackID string) (string, error) {
      query := `SELECT content FROM track_lyrics WHERE track_id = $1`
      var content string
      err := r.db.QueryRowContext(ctx, query, trackID).Scan(&content)
      return content, err
  }
  ```

- [ ] **Step 3: Create tests for postgres repository**
  Add mock tests in `services/music-service/internal/repository/postgres_test.go`. Ensure it saves and loads lyrics correctly.
  Run: `go test -v ./services/music-service/internal/repository/...`
  Expected: PASS

- [ ] **Step 4: Commit**
  ```bash
  git add services/music-service/internal/domain/models.go services/music-service/internal/repository/postgres.go services/music-service/internal/repository/postgres_test.go
  git commit -m "feat(music): implement lyrics repository storage and tests"
  ```

---

### Task 3: REST API for Lyrics (`music-service`)

**Files:**
- Modify: `services/music-service/internal/usecase/music_usecase.go`
- Modify: `services/music-service/internal/delivery/http/handler.go`

**Interfaces:**
- Consumes: `SaveLyrics` and `GetLyrics` from postgres repository.
- Produces: HTTP Endpoints:
  - `GET /api/v1/music/tracks/:track_id/lyrics`
  - `POST /api/v1/music/tracks/:track_id/lyrics`

- [ ] **Step 1: Add lyrics methods in Usecase**
  Modify usecase interface and add implementation in `services/music-service/internal/usecase/music_usecase.go`:
  ```go
  // Add to Usecase:
  SaveLyrics(ctx context.Context, trackID string, content string) error
  GetLyrics(ctx context.Context, trackID string) (string, error)
  ```

- [ ] **Step 2: Add REST controllers in delivery handler**
  Implement HTTP functions in `services/music-service/internal/delivery/http/handler.go`:
  ```go
  func (h *Handler) GetLyrics(c *gin.Context) {
      trackID := c.Param("track_id")
      content, err := h.uc.GetLyrics(c.Request.Context(), trackID)
      if err != nil {
          c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "LYRICS_NOT_FOUND", "message": "Chưa có lời."}})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"track_id": trackID, "content": content}})
  }

  func (h *Handler) SaveLyrics(c *gin.Context) {
      trackID := c.Param("track_id")
      var req struct {
          Content string `json:"content" binding:"required"`
      }
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
          return
      }
      if err := h.uc.SaveLyrics(c.Request.Context(), trackID, req.Content); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Cập nhật thành công."}})
  }
  ```
  Register routes:
  ```go
  r.GET("/tracks/:track_id/lyrics", handler.GetLyrics)
  r.POST("/tracks/:track_id/lyrics", handler.SaveLyrics)
  ```

- [ ] **Step 3: Test HTTP endpoints via curl**
  Run: `curl -i http://localhost:8080/api/v1/music/tracks/mock-track-id/lyrics`
  Expected: HTTP 404 (or 200 if seeded).

- [ ] **Step 4: Commit**
  ```bash
  git add services/music-service/internal/usecase/music_usecase.go services/music-service/internal/delivery/http/handler.go
  git commit -m "feat(music): implement lyrics REST API endpoints"
  ```

---

### Task 4: Repository implementation for Bookmarks (`music-service`)

**Files:**
- Modify: `services/music-service/internal/domain/models.go`
- Modify: `services/music-service/internal/repository/postgres.go`

**Interfaces:**
- Consumes: Postgres DB connection
- Produces:
  ```go
  SaveBookmark(ctx context.Context, b *domain.Bookmark) error
  GetBookmarks(ctx context.Context, roomID string) ([]*domain.Bookmark, error)
  DeleteBookmark(ctx context.Context, id string) error
  ```

- [ ] **Step 1: Add Bookmark struct in models.go**
  Add structures in `services/music-service/internal/domain/models.go`:
  ```go
  type Bookmark struct {
      ID         string    `json:"id"`
      RoomID     string    `json:"room_id"`
      UserID     string    `json:"user_id"`
      TrackID    string    `json:"track_id"`
      PositionMS int       `json:"position_ms"`
      Note       string    `json:"note"`
      CreatedAt  time.Time `json:"created_at"`
  }
  ```

- [ ] **Step 2: Implement postgres repo methods**
  Add implementation at the end of `services/music-service/internal/repository/postgres.go`:
  ```go
  func (r *PostgresRepository) SaveBookmark(ctx context.Context, b *domain.Bookmark) error {
      query := `INSERT INTO bookmarks (id, room_id, user_id, track_id, position_ms, note) VALUES ($1, $2, $3, $4, $5, $6)`
      _, err := r.db.ExecContext(ctx, query, b.ID, b.RoomID, b.UserID, b.TrackID, b.PositionMS, b.Note)
      return err
  }

  func (r *PostgresRepository) GetBookmarks(ctx context.Context, roomID string) ([]*domain.Bookmark, error) {
      query := `SELECT id, room_id, user_id, track_id, position_ms, note, created_at FROM bookmarks WHERE room_id = $1 ORDER BY created_at DESC`
      rows, err := r.db.QueryContext(ctx, query, roomID)
      if err != nil {
          return nil, err
      }
      defer rows.Close()

      var list []*domain.Bookmark
      for rows.Next() {
          var b domain.Bookmark
          if err := rows.Scan(&b.ID, &b.RoomID, &b.UserID, &b.TrackID, &b.PositionMS, &b.Note, &b.CreatedAt); err != nil {
              return nil, err
          }
          list = append(list, &b)
      }
      return list, nil
  }

  func (r *PostgresRepository) DeleteBookmark(ctx context.Context, id string) error {
      _, err := r.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE id = $1`, id)
      return err
  }
  ```

- [ ] **Step 3: Run repository test suite**
  Run: `go test -v ./services/music-service/internal/repository/...`
  Expected: PASS

- [ ] **Step 4: Commit**
  ```bash
  git add services/music-service/internal/domain/models.go services/music-service/internal/repository/postgres.go
  git commit -m "feat(music): implement bookmarks repository database methods"
  ```

---

### Task 5: REST API for Bookmarks (`music-service`)

**Files:**
- Modify: `services/music-service/internal/usecase/music_usecase.go`
- Modify: `services/music-service/internal/delivery/http/handler.go`

**Interfaces:**
- Consumes: Bookmarks postgres database methods.
- Produces: HTTP endpoints:
  - `POST /api/v1/music/rooms/:room_id/bookmarks`
  - `GET /api/v1/music/rooms/:room_id/bookmarks`
  - `DELETE /api/v1/music/rooms/:room_id/bookmarks/:id`

- [ ] **Step 1: Update usecase for bookmarks**
  Modify interfaces and implementation in `services/music-service/internal/usecase/music_usecase.go` to add `SaveBookmark`, `GetBookmarks`, and `DeleteBookmark`.

- [ ] **Step 2: Add REST controllers in delivery handler**
  Implement HTTP functions in `services/music-service/internal/delivery/http/handler.go`:
  ```go
  func (h *Handler) GetBookmarks(c *gin.Context) {
      roomID := c.Param("room_id")
      list, err := h.uc.GetBookmarks(c.Request.Context(), roomID)
      if err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
  }

  func (h *Handler) SaveBookmark(c *gin.Context) {
      roomID := c.Param("room_id")
      userID := c.GetString("user_id") // extracted from auth middleware
      var req struct {
          TrackID    string `json:"track_id" binding:"required"`
          PositionMS int    `json:"position_ms" binding:"required"`
          Note       string `json:"note" binding:"required"`
      }
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
          return
      }
      b := &domain.Bookmark{
          ID:         uuid.NewString(), // import github.com/google/uuid
          RoomID:     roomID,
          UserID:     userID,
          TrackID:    req.TrackID,
          PositionMS: req.PositionMS,
          Note:       req.Note,
      }
      if err := h.uc.SaveBookmark(c.Request.Context(), b); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusCreated, gin.H{"success": true, "data": b})
  }

  func (h *Handler) DeleteBookmark(c *gin.Context) {
      id := c.Param("id")
      if err := h.uc.DeleteBookmark(c.Request.Context(), id); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Đã xóa bookmark thành công."}})
  }
  ```
  Register routes:
  ```go
  r.GET("/rooms/:room_id/bookmarks", handler.GetBookmarks)
  r.POST("/rooms/:room_id/bookmarks", handler.SaveBookmark)
  r.DELETE("/rooms/:room_id/bookmarks/:id", handler.DeleteBookmark)
  ```

- [ ] **Step 3: Test REST APIs with curl**
  Run: `curl -i http://localhost:8080/api/v1/music/rooms/mock-room-id/bookmarks`
  Expected: HTTP 401 Unauthorized (since it requires Bearer token) or HTTP 200 with JWT.

- [ ] **Step 4: Commit**
  ```bash
  git add services/music-service/internal/usecase/music_usecase.go services/music-service/internal/delivery/http/handler.go
  git commit -m "feat(music): add bookmarks REST HTTP endpoints"
  ```

---

### Task 6: DJ Mode/Takeover Backend (`playback-service`)

**Files:**
- Modify: `services/playback-service/internal/repository/redis.go`
- Modify: `services/playback-service/internal/usecase/playback_usecase.go`
- Modify: `services/playback-service/internal/delivery/http/handler.go` (or `main.go`)

**Interfaces:**
- Consumes: Redis client connection.
- Produces:
  - Redis key `room:<room_id>:guest_dj` with TTL.
  - HTTP `POST /api/v1/rooms/:room_id/playback/dj` and `DELETE /api/v1/rooms/:room_id/playback/dj`
  - Playback WS control filtering validation.

- [ ] **Step 1: Implement Redis DJ methods**
  Add functions in `services/playback-service/internal/repository/redis.go` (or usecase):
  ```go
  func (r *RedisRepository) SetGuestDJ(ctx context.Context, roomID string, userID string, ttl time.Duration) error {
      return r.rdb.Set(ctx, "room:"+roomID+":guest_dj", userID, ttl).Err()
  }

  func (r *RedisRepository) GetGuestDJ(ctx context.Context, roomID string) (string, error) {
      val, err := r.rdb.Get(ctx, "room:"+roomID+":guest_dj").Result()
      if err == redis.Nil {
          return "", nil
      }
      return val, err
  }

  func (r *RedisRepository) ClearGuestDJ(ctx context.Context, roomID string) error {
      return r.rdb.Del(ctx, "room:"+roomID+":guest_dj").Err()
  }
  ```

- [ ] **Step 2: Add REST controller for Guest DJ**
  Expose routes in `cmd/server/main.go` under `playbackGroup`:
  ```go
  playbackGroup.POST("/dj", func(c *gin.Context) {
      roomID := c.Param("room_id")
      var req struct {
          UserID          string `json:"user_id" binding:"required"`
          DurationSeconds int    `json:"duration_seconds" binding:"required"`
      }
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
          return
      }
      // Save DJ state to Redis
      if err := repo.SetGuestDJ(c.Request.Context(), roomID, req.UserID, time.Duration(req.DurationSeconds)*time.Second); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      // Broadcast takeover via Hub
      hub.BroadcastToRoom(roomID, gin.H{
          "event": "dj:takeover",
          "payload": gin.H{
              "user_id":      req.UserID,
              "ends_at":      time.Now().Add(time.Duration(req.DurationSeconds) * time.Second).UnixMilli(),
          },
      })
      c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Đã nhường quyền Guest DJ."}})
  })

  playbackGroup.DELETE("/dj", func(c *gin.Context) {
      roomID := c.Param("room_id")
      if err := repo.ClearGuestDJ(c.Request.Context(), roomID); err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
          return
      }
      hub.BroadcastToRoom(roomID, gin.H{
          "event": "dj:released",
          "payload": gin.H{
              "reason": "revoked",
          },
      })
      c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Đã thu hồi quyền Guest DJ."}})
  })
  ```

- [ ] **Step 3: Add WS permissions check**
  In WS event handler (e.g. `ServePlaybackWS` or `Hub` client write loop), when `playback:control` is received:
  Check if `guest_dj` is active. If so, and the sender is neither Host nor `guest_dj`, reply with an error message and ignore the command.

- [ ] **Step 4: Commit**
  ```bash
  git add services/playback-service/internal/repository/redis.go services/playback-service/cmd/server/main.go
  git commit -m "feat(playback): implement Guest DJ role takeover backend"
  ```

---

### Task 7: Realtime Up-Next Song Poll Backend (`playback-service`)

**Files:**
- Modify: `services/playback-service/internal/delivery/http/hub.go`
- Modify: `services/playback-service/internal/usecase/playback_usecase.go`

**Interfaces:**
- Consumes: WS Connection loop.
- Produces: WS Events `poll:start`, `poll:update`, `poll:end` and up-next prioritization.

- [ ] **Step 1: Add check loop for last 30s of song**
  In the playback synchronization loop in `usecase/playback_usecase.go`, periodically check current position vs track duration:
  - If duration - position <= 30000ms AND poll not yet started:
    - Set Redis key `room:<room_id>:poll_active = true`.
    - Fetch top 3 songs in room playlist/queue.
    - Broadcast `poll:start` through Hub.

- [ ] **Step 2: Add WS vote handler**
  In the WS write/read pump:
  - Add handler for event `poll:vote`:
    - Increment vote for candidate track in Redis.
    - Calculate and broadcast `poll:update` containing vote counts.

- [ ] **Step 3: Enforce winner on track end**
  When the track ends:
  - Fetch vote map from Redis, determine track with max votes.
  - Re-order queue: Move the winning track to the head of the queue.
  - Clear Redis keys `poll_active` and vote details.
  - Broadcast `poll:end` and trigger next track play.

- [ ] **Step 4: Commit**
  ```bash
  git add services/playback-service/internal/delivery/http/hub.go services/playback-service/internal/usecase/playback_usecase.go
  git commit -m "feat(playback): implement up-next song poll backend"
  ```

---

### Task 8: Web Client API & Service integration

**Files:**
- Create: `web-client/src/app/core/services/lyrics.service.ts`
- Create: `web-client/src/app/core/services/bookmarks.service.ts`
- Modify: `web-client/src/app/core/services/playback.service.ts`

**Interfaces:**
- Consumes: HTTP client, room parameters.
- Produces: APIs to get/save lyrics, manage room bookmarks, and WS events mapping.

- [ ] **Step 1: Implement LyricsService**
  Create `web-client/src/app/core/services/lyrics.service.ts`:
  ```typescript
  @Injectable({ providedIn: 'root' })
  export class LyricsService {
    private http = inject(HttpClient);
    getLyrics(trackId: string) {
      return this.http.get<any>(`/api/v1/music/tracks/${trackId}/lyrics`);
    }
    saveLyrics(trackId: string, content: string) {
      return this.http.post<any>(`/api/v1/music/tracks/${trackId}/lyrics`, { content });
    }
  }
  ```

- [ ] **Step 2: Implement BookmarksService**
  Create `web-client/src/app/core/services/bookmarks.service.ts`:
  ```typescript
  @Injectable({ providedIn: 'root' })
  export class BookmarksService {
    private http = inject(HttpClient);
    getBookmarks(roomId: string) {
      return this.http.get<any>(`/api/v1/music/rooms/${roomId}/bookmarks`);
    }
    saveBookmark(roomId: string, trackId: string, positionMs: number, note: string) {
      return this.http.post<any>(`/api/v1/music/rooms/${roomId}/bookmarks`, { track_id: trackId, position_ms: positionMs, note });
    }
    deleteBookmark(roomId: string, id: string) {
      return this.http.delete<any>(`/api/v1/music/rooms/${roomId}/bookmarks/${id}`);
    }
  }
  ```

- [ ] **Step 3: Update PlaybackService WS handlers**
  In `PlaybackService`, listen to WS events `dj:takeover`, `dj:released`, `poll:start`, `poll:update`, `poll:end` and emit them to corresponding RxJS BehaviorSubjects.

- [ ] **Step 4: Commit**
  ```bash
  git add web-client/src/app/core/services/lyrics.service.ts web-client/src/app/core/services/bookmarks.service.ts web-client/src/app/core/services/playback.service.ts
  git commit -m "feat(client): implement lyrics and bookmarks HTTP services"
  ```

---

### Task 9: Web Client Synced Lyrics Component

**Files:**
- Create: `web-client/src/app/features/room/components/lyrics/lyrics.component.ts`
- Create: `web-client/src/app/features/room/components/lyrics/lyrics.component.html`
- Create: `web-client/src/app/features/room/components/lyrics/lyrics.component.css`

**Interfaces:**
- Consumes: `LyricsService` and playback position subscription.
- Produces: Synced lyrics display panel and editing modal.

- [ ] **Step 1: Create HTML structure**
  Add lines list showing LRC lyrics:
  ```html
  <div class="lyrics-container">
    @if (lyricsLines.length > 0) {
      <div class="lines-list">
        @for (line of lyricsLines; track $index) {
          <div class="lyric-line" [class.active]="$index === activeIndex" (click)="seekTo(line.timeMs)">
            {{ line.text }}
          </div>
        }
      </div>
    } @else {
      <div class="no-lyrics">
        Chưa có lời bài hát. <button (click)="openEditModal()">Thêm lời (.lrc)</button>
      </div>
    }
  </div>
  ```

- [ ] **Step 2: Implement LRC Parser and progress auto-scroll**
  In TS, parse LRC using regex `\[(\d+):(\d+)\.(\d+)\](.*)` and compute timestamps. Use `scroll-into-view` logic to auto-scroll active lyric line to center.

- [ ] **Step 3: Add unit tests**
  Add unit tests in `lyrics.component.spec.ts` mocking the parser.
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS

- [ ] **Step 4: Commit**
  ```bash
  git add web-client/src/app/features/room/components/lyrics/
  git commit -m "feat(client): create Synced Lyrics UI component and tests"
  ```

---

### Task 10: Web Client Bookmark & Chat Integration

**Files:**
- Create: `web-client/src/app/features/room/components/bookmarks/bookmarks.component.ts`
- Modify: `web-client/src/app/features/room/components/chat/chat.component.ts`

**Interfaces:**
- Consumes: `BookmarksService`, chat messages format.
- Produces: Bookmark creation form, list, and clickable seek links inside Chat view.

- [ ] **Step 1: Implement Bookmarks tab**
  In `bookmarks.component.ts`, let users click a bookmark icon to open note input, save to DB, and render list with a click-to-seek callback.

- [ ] **Step 2: Implement click-to-seek parser in Chat**
  In `chat.component.ts`, when rendering messages, parse text matching `\[(\d+):(\d+)\]`. If match, replace it with a clickable `<button>` calling `playbackService.seek(timeMs)`.

- [ ] **Step 3: Run client tests**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS

- [ ] **Step 4: Commit**
  ```bash
  git add web-client/src/app/features/room/components/bookmarks/ web-client/src/app/features/room/components/chat/chat.component.ts
  git commit -m "feat(client): integrate bookmarks with chat seek actions"
  ```

---

### Task 11: Web Client Realtime Poll & DJ UI

**Files:**
- Create: `web-client/src/app/features/room/components/poll-widget/poll-widget.component.ts`
- Modify: `web-client/src/app/features/room/components/sidebar/sidebar.component.ts`

**Interfaces:**
- Consumes: Playback WS subjects.
- Produces: Realtime vote sliders overlay and locked playback controls banner.

- [ ] **Step 1: Implement PollWidgetComponent**
  Render floating overlay when `poll:start` is received, displaying progress bar and candidates. On option click, emit `poll:vote` WS command.

- [ ] **Step 2: Add Guest DJ locking in controls**
  In room controls, if a guest DJ is active, lock playback buttons for other members and display banner `[User] đang làm Guest DJ`.

- [ ] **Step 3: Run all unit tests**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: 100% PASS

- [ ] **Step 4: Commit**
  ```bash
  git add web-client/src/app/features/room/components/poll-widget/ web-client/src/app/features/room/components/sidebar/sidebar.component.ts
  git commit -m "feat(client): implement poll widget overlay and guest DJ UI controls"
  ```
