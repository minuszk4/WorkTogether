package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/worktogether/services/playlist-service/internal/domain"
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

	// Khởi tạo bảng playlists
	playlistsSchema := `
	CREATE TABLE IF NOT EXISTS playlists (
		id VARCHAR(36) PRIMARY KEY,
		room_id VARCHAR(36),
		user_id VARCHAR(36),
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.ExecContext(ctx, playlistsSchema); err != nil {
		return err
	}

	// Khởi tạo bảng playlist_tracks
	tracksSchema := `
	CREATE TABLE IF NOT EXISTS playlist_tracks (
		id VARCHAR(36) PRIMARY KEY,
		playlist_id VARCHAR(36) NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
		track_id VARCHAR(36) NOT NULL,
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255),
		thumbnail_url TEXT,
		duration_ms INTEGER NOT NULL,
		source_url TEXT NOT NULL,
		position INTEGER NOT NULL,
		added_by VARCHAR(36) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.ExecContext(ctx, tracksSchema); err != nil {
		return err
	}

	// Khởi tạo bảng playlist_votes
	votesSchema := `
	CREATE TABLE IF NOT EXISTS playlist_votes (
		playlist_track_id VARCHAR(36) NOT NULL REFERENCES playlist_tracks(id) ON DELETE CASCADE,
		user_id VARCHAR(36) NOT NULL,
		vote_type VARCHAR(10) NOT NULL, -- "up" or "down"
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (playlist_track_id, user_id)
	);`
	if _, err := r.db.ExecContext(ctx, votesSchema); err != nil {
		return err
	}

	log.Println("Đã khởi tạo schema PostgreSQL cho playlist-service thành công.")
	return nil
}

func (r *PostgresRepository) CreatePlaylist(ctx context.Context, p *domain.Playlist) error {
	query := `INSERT INTO playlists (id, room_id, user_id, name, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.RoomID, p.UserID, p.Name, p.CreatedAt)
	return err
}

func (r *PostgresRepository) GetPlaylistByID(ctx context.Context, id string) (*domain.Playlist, error) {
	query := `SELECT id, room_id, user_id, name, created_at FROM playlists WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var p domain.Playlist
	err := row.Scan(&p.ID, &p.RoomID, &p.UserID, &p.Name, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) GetRoomPlaylists(ctx context.Context, roomID string) ([]*domain.Playlist, error) {
	query := `SELECT id, room_id, user_id, name, created_at FROM playlists WHERE room_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*domain.Playlist
	for rows.Next() {
		var p domain.Playlist
		if err := rows.Scan(&p.ID, &p.RoomID, &p.UserID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, &p)
	}
	return playlists, nil
}

func (r *PostgresRepository) GetUserPlaylists(ctx context.Context, userID string) ([]*domain.Playlist, error) {
	query := `SELECT id, room_id, user_id, name, created_at FROM playlists WHERE user_id = $1 AND room_id IS NULL ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*domain.Playlist
	for rows.Next() {
		var p domain.Playlist
		if err := rows.Scan(&p.ID, &p.RoomID, &p.UserID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, &p)
	}
	return playlists, nil
}

func (r *PostgresRepository) DeletePlaylist(ctx context.Context, id string) error {
	query := `DELETE FROM playlists WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) AddTrack(ctx context.Context, t *domain.PlaylistTrack) error {
	// Tính toán position kế tiếp
	posQuery := `SELECT COALESCE(MAX(position), -1) + 1 FROM playlist_tracks WHERE playlist_id = $1`
	err := r.db.QueryRowContext(ctx, posQuery, t.PlaylistID).Scan(&t.Position)
	if err != nil {
		return err
	}

	query := `INSERT INTO playlist_tracks (id, playlist_id, track_id, title, artist, thumbnail_url, duration_ms, source_url, position, added_by, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = r.db.ExecContext(ctx, query, t.ID, t.PlaylistID, t.TrackID, t.Title, t.Artist, t.Thumbnail, t.DurationMS, t.SourceURL, t.Position, t.AddedBy, t.CreatedAt)
	return err
}

func (r *PostgresRepository) RemoveTrack(ctx context.Context, playlistID, trackItemID string) error {
	// Bắt đầu Transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lấy position của bài hát cần xóa
	var pos int
	posQuery := `SELECT position FROM playlist_tracks WHERE id = $1 AND playlist_id = $2`
	err = tx.QueryRowContext(ctx, posQuery, trackItemID, playlistID).Scan(&pos)
	if err != nil {
		return err
	}

	// Xóa bài hát
	deleteQuery := `DELETE FROM playlist_tracks WHERE id = $1`
	_, err = tx.ExecContext(ctx, deleteQuery, trackItemID)
	if err != nil {
		return err
	}

	// Cập nhật lại position của các bài hát phía sau
	updateQuery := `UPDATE playlist_tracks SET position = position - 1 WHERE playlist_id = $1 AND position > $2`
	_, err = tx.ExecContext(ctx, updateQuery, playlistID, pos)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) MoveTrack(ctx context.Context, playlistID, trackItemID string, newPos int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldPos int
	posQuery := `SELECT position FROM playlist_tracks WHERE id = $1 AND playlist_id = $2`
	err = tx.QueryRowContext(ctx, posQuery, trackItemID, playlistID).Scan(&oldPos)
	if err != nil {
		return err
	}

	if oldPos == newPos {
		return nil
	}

	if oldPos < newPos {
		// Dịch các bài ở giữa lên trên
		shiftQuery := `UPDATE playlist_tracks SET position = position - 1 WHERE playlist_id = $1 AND position > $2 AND position <= $3`
		_, err = tx.ExecContext(ctx, shiftQuery, playlistID, oldPos, newPos)
		if err != nil {
			return err
		}
	} else {
		// Dịch các bài ở giữa xuống dưới
		shiftQuery := `UPDATE playlist_tracks SET position = position + 1 WHERE playlist_id = $1 AND position >= $2 AND position < $3`
		_, err = tx.ExecContext(ctx, shiftQuery, playlistID, newPos, oldPos)
		if err != nil {
			return err
		}
	}

	// Đặt bài hát vào vị trí mới
	setQuery := `UPDATE playlist_tracks SET position = $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, setQuery, newPos, trackItemID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*domain.PlaylistTrack, error) {
	// Lấy danh sách bài hát cùng số vote tương ứng
	query := `
		SELECT t.id, t.playlist_id, t.track_id, t.title, t.artist, t.thumbnail_url, t.duration_ms, t.source_url, t.position, t.added_by, t.created_at,
		       COALESCE(SUM(CASE WHEN v.vote_type = 'up' THEN 1 WHEN v.vote_type = 'down' THEN -1 ELSE 0 END), 0) as votes
		FROM playlist_tracks t
		LEFT JOIN playlist_votes v ON t.id = v.playlist_track_id
		WHERE t.playlist_id = $1
		GROUP BY t.id
		ORDER BY votes DESC, t.position ASC`

	rows, err := r.db.QueryContext(ctx, query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*domain.PlaylistTrack
	for rows.Next() {
		var t domain.PlaylistTrack
		var artist sql.NullString
		var thumb sql.NullString
		err := rows.Scan(&t.ID, &t.PlaylistID, &t.TrackID, &t.Title, &artist, &thumb, &t.DurationMS, &t.SourceURL, &t.Position, &t.AddedBy, &t.CreatedAt, &t.Votes)
		if err != nil {
			return nil, err
		}
		t.Artist = artist.String
		t.Thumbnail = thumb.String
		tracks = append(tracks, &t)
	}
	return tracks, nil
}

func (r *PostgresRepository) SaveVote(ctx context.Context, v *domain.PlaylistVote) error {
	query := `
		INSERT INTO playlist_votes (playlist_track_id, user_id, vote_type, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (playlist_track_id, user_id) DO UPDATE
		SET vote_type = EXCLUDED.vote_type, created_at = EXCLUDED.created_at`
	_, err := r.db.ExecContext(ctx, query, v.PlaylistTrackID, v.UserID, v.VoteType, v.CreatedAt)
	return err
}

func (r *PostgresRepository) DeleteVote(ctx context.Context, trackItemID, userID string) error {
	query := `DELETE FROM playlist_votes WHERE playlist_track_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, trackItemID, userID)
	return err
}
