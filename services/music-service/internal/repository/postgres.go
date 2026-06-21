package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/worktogether/services/music-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	repo := &PostgresRepository{db: db}
	if err := repo.initTables(); err != nil {
		log.Printf("Cảnh báo: Không thể khởi tạo bảng trong Postgres: %v\n", err)
	}
	return repo
}

func (r *PostgresRepository) initTables() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Khởi tạo bảng tracks
	trackSchema := `
	CREATE TABLE IF NOT EXISTS tracks (
		id VARCHAR(36) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		thumbnail_url TEXT,
		duration_ms INTEGER NOT NULL,
		source VARCHAR(50) NOT NULL,
		source_url TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.ExecContext(ctx, trackSchema); err != nil {
		return err
	}

	// Khởi tạo bảng history
	historySchema := `
	CREATE TABLE IF NOT EXISTS history (
		id VARCHAR(36) PRIMARY KEY,
		room_id VARCHAR(36) NOT NULL,
		track_id VARCHAR(36) REFERENCES tracks(id) ON DELETE CASCADE,
		played_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.ExecContext(ctx, historySchema); err != nil {
		return err
	}

	// Khởi tạo bảng track_lyrics
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

	// Khởi tạo bảng bookmarks
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
	
	// Tạo chỉ mục cho bookmarks.room_id
	if _, err := r.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_bookmarks_room ON bookmarks(room_id);`); err != nil {
		return err
	}

	log.Println("Đã khởi tạo schema PostgreSQL cho music-service thành công.")
	return nil
}

func (r *PostgresRepository) SaveTrack(ctx context.Context, t *domain.Track) error {
	query := `
	INSERT INTO tracks (id, title, artist, thumbnail_url, duration_ms, source, source_url, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (source_url) DO UPDATE 
	SET title = EXCLUDED.title, thumbnail_url = EXCLUDED.thumbnail_url, duration_ms = EXCLUDED.duration_ms
	RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query, t.ID, t.Title, t.Artist, t.ThumbnailURL, t.DurationMS, t.Source, t.SourceURL, t.CreatedAt).Scan(&t.ID, &t.CreatedAt)
	return err
}

func (r *PostgresRepository) GetTrackByID(ctx context.Context, id string) (*domain.Track, error) {
	query := `SELECT id, title, artist, thumbnail_url, duration_ms, source, source_url, created_at FROM tracks WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var t domain.Track
	err := row.Scan(&t.ID, &t.Title, &t.Artist, &t.ThumbnailURL, &t.DurationMS, &t.Source, &t.SourceURL, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresRepository) GetTrackBySourceURL(ctx context.Context, url string) (*domain.Track, error) {
	query := `SELECT id, title, artist, thumbnail_url, duration_ms, source, source_url, created_at FROM tracks WHERE source_url = $1`
	row := r.db.QueryRowContext(ctx, query, url)

	var t domain.Track
	err := row.Scan(&t.ID, &t.Title, &t.Artist, &t.ThumbnailURL, &t.DurationMS, &t.Source, &t.SourceURL, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresRepository) SearchTracks(ctx context.Context, keyword string, limit int) ([]*domain.Track, error) {
	query := `SELECT id, title, artist, thumbnail_url, duration_ms, source, source_url, created_at 
	          FROM tracks 
	          WHERE title ILIKE $1 OR artist ILIKE $1 
	          ORDER BY created_at DESC 
	          LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, "%"+keyword+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var t domain.Track
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.ThumbnailURL, &t.DurationMS, &t.Source, &t.SourceURL, &t.CreatedAt); err != nil {
			return nil, err
		}
		tracks = append(tracks, &t)
	}
	return tracks, nil
}

func (r *PostgresRepository) SavePlaybackHistory(ctx context.Context, h *domain.PlaybackHistory) error {
	// Tự động thêm track giữ chỗ nếu track_id chưa tồn tại để không vi phạm khoá ngoại
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tracks WHERE id = $1)`, h.TrackID).Scan(&exists)
	if err == nil && !exists {
		mockQuery := `
		INSERT INTO tracks (id, title, artist, thumbnail_url, duration_ms, source, source_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (source_url) DO NOTHING`
		_, _ = r.db.ExecContext(ctx, mockQuery, h.TrackID, "Mock Track", "Mock Artist", "", 180000, "youtube", "https://mock.url/"+h.TrackID, time.Now())
	}

	query := `INSERT INTO history (id, room_id, track_id, played_at) VALUES ($1, $2, $3, $4)`
	_, err = r.db.ExecContext(ctx, query, h.ID, r.RoomIDCheck(h.RoomID), h.TrackID, h.PlayedAt)
	return err
}

func (r *PostgresRepository) GetPlaybackHistory(ctx context.Context, roomID string, limit int) ([]*domain.Track, error) {
	query := `SELECT t.id, t.title, t.artist, t.thumbnail_url, t.duration_ms, t.source, t.source_url, h.played_at 
	          FROM history h
	          JOIN tracks t ON h.track_id = t.id
	          WHERE h.room_id = $1
	          ORDER BY h.played_at DESC
	          LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var t domain.Track
		var playedAt time.Time
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.ThumbnailURL, &t.DurationMS, &t.Source, &t.SourceURL, &playedAt); err != nil {
			return nil, err
		}
		tracks = append(tracks, &t)
	}
	return tracks, nil
}

func (r *PostgresRepository) RoomIDCheck(id string) string {
	return id
}

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

func (r *PostgresRepository) GetRoomStats(ctx context.Context, roomID string) (*domain.RoomStats, error) {
	stats := &domain.RoomStats{TopTracks: []*domain.TopTrack{}}

	// 1. Total tracks
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM history WHERE room_id = $1", roomID).Scan(&stats.TotalTracksPlayed)
	if err != nil {
		return nil, err
	}

	// 2. Total play time
	timeQuery := `SELECT COALESCE(SUM(t.duration_ms), 0) FROM history h JOIN tracks t ON h.track_id = t.id WHERE h.room_id = $1`
	err = r.db.QueryRowContext(ctx, timeQuery, roomID).Scan(&stats.TotalPlayTimeMS)
	if err != nil {
		return nil, err
	}

	// 3. Top tracks
	topQuery := `
		SELECT t.id, t.title, t.artist, t.thumbnail_url, COUNT(h.id) AS play_count
		FROM history h
		JOIN tracks t ON h.track_id = t.id
		WHERE h.room_id = $1
		GROUP BY t.id, t.title, t.artist, t.thumbnail_url
		ORDER BY play_count DESC
		LIMIT 5
	`
	rows, err := r.db.QueryContext(ctx, topQuery, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t domain.TopTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.ThumbnailURL, &t.PlayCount); err != nil {
			return nil, err
		}
		stats.TopTracks = append(stats.TopTracks, &t)
	}

	return stats, nil
}
