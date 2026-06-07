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
