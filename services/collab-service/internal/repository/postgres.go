package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/worktogether/services/collab-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetOrCreateNote(ctx context.Context, roomID string) (*domain.Note, []*domain.NoteBlock, error) {
	// 1. Get note
	query := `SELECT id, room_id, title, created_at, updated_at FROM notes WHERE room_id = $1`
	var note domain.Note
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&note.ID, &note.RoomID, &note.Title, &note.CreatedAt, &note.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default note
		noteID := roomID // We can use roomID as NoteID for 1-to-1 room-note mapping, which is extremely simple and elegant!
		// Or generate a uuid
		insertQuery := `INSERT INTO notes (id, room_id, title, created_at, updated_at) VALUES ($1, $1, $2, NOW(), NOW()) RETURNING id, room_id, title, created_at, updated_at`
		err = r.db.QueryRowContext(ctx, insertQuery, noteID, "Ghi chú phòng").Scan(&note.ID, &note.RoomID, &note.Title, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	}

	// 2. Get blocks
	blocksQuery := `SELECT id, note_id, block_type, content, is_checked, order_index, updated_by, updated_at FROM note_blocks WHERE note_id = $1 ORDER BY order_index ASC`
	rows, err := r.db.QueryContext(ctx, blocksQuery, note.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var blocks []*domain.NoteBlock
	for rows.Next() {
		var b domain.NoteBlock
		if err := rows.Scan(&b.ID, &b.NoteID, &b.BlockType, &b.Content, &b.IsChecked, &b.OrderIndex, &b.UpdatedBy, &b.UpdatedAt); err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, &b)
	}

	return &note, blocks, nil
}

func (r *PostgresRepository) AddBlock(ctx context.Context, noteID string, b *domain.NoteBlock) error {
	query := `INSERT INTO note_blocks (id, note_id, block_type, content, is_checked, order_index, updated_by, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	          RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query, b.ID, noteID, b.BlockType, b.Content, b.IsChecked, b.OrderIndex, b.UpdatedBy).Scan(&b.UpdatedAt)
}

func (r *PostgresRepository) UpdateBlock(ctx context.Context, noteID string, b *domain.NoteBlock) error {
	query := `UPDATE note_blocks 
	          SET content = $1, is_checked = $2, updated_by = $3, updated_at = NOW() 
	          WHERE id = $4 AND note_id = $5 
	          RETURNING updated_at`
	err := r.db.QueryRowContext(ctx, query, b.Content, b.IsChecked, b.UpdatedBy, b.ID, noteID).Scan(&b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("block not found")
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteBlock(ctx context.Context, noteID string, blockID string) error {
	query := `DELETE FROM note_blocks WHERE id = $1 AND note_id = $2`
	res, err := r.db.ExecContext(ctx, query, blockID, noteID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("block not found")
	}
	return nil
}

func (r *PostgresRepository) UpdateBlocksOrder(ctx context.Context, noteID string, blockIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for idx, id := range blockIDs {
		query := `UPDATE note_blocks SET order_index = $1 WHERE id = $2 AND note_id = $3`
		_, err := tx.ExecContext(ctx, query, idx, id, noteID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetWhiteboardSnapshot(ctx context.Context, roomID string) (*domain.WhiteboardSnapshot, error) {
	snapshot := &domain.WhiteboardSnapshot{RoomID: roomID}
	err := r.db.QueryRowContext(ctx, `SELECT snapshot, updated_at FROM whiteboard_snapshots WHERE room_id = $1`, roomID).Scan(&snapshot.Snapshot, &snapshot.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (r *PostgresRepository) SaveWhiteboardSnapshot(ctx context.Context, roomID string, snapshot []byte) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO whiteboard_snapshots (room_id, snapshot, updated_at)
		VALUES ($1, $2::jsonb, NOW())
		ON CONFLICT (room_id) DO UPDATE SET snapshot = EXCLUDED.snapshot, updated_at = NOW()`, roomID, snapshot)
	return err
}
