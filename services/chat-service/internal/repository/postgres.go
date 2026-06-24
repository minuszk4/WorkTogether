package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/worktogether/services/chat-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (id, room_id, sender_id, content, reply_to_id, is_edited, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var replyTo interface{} = msg.ReplyToID
	if msg.ReplyToID == "" {
		replyTo = nil
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.RoomID, msg.SenderID, msg.Content, replyTo, msg.IsEdited, msg.CreatedAt)
	return err
}

func (r *PostgresRepository) GetMessageByID(ctx context.Context, id string) (*domain.Message, error) {
	query := `SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1`
	msg := &domain.Message{}
	var replyTo sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Content, &replyTo, &msg.IsEdited, &msg.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if replyTo.Valid {
		msg.ReplyToID = replyTo.String
	}
	return msg, nil
}

func (r *PostgresRepository) GetMessagesByRoom(ctx context.Context, roomID string, beforeID string, limit int) ([]*domain.Message, error) {
	var query string
	var err error
	var rows *sql.Rows

	if beforeID != "" {
		var beforeTime time.Time
		err = r.db.QueryRowContext(ctx, "SELECT created_at FROM messages WHERE id = $1", beforeID).Scan(&beforeTime)
		if err != nil {
			return nil, err
		}

		query = `
			SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at 
			FROM messages 
			WHERE room_id = $1 AND created_at < $2 
			ORDER BY created_at DESC 
			LIMIT $3
		`
		rows, err = r.db.QueryContext(ctx, query, roomID, beforeTime, limit)
	} else {
		query = `
			SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at 
			FROM messages 
			WHERE room_id = $1 
			ORDER BY created_at DESC 
			LIMIT $2
		`
		rows, err = r.db.QueryContext(ctx, query, roomID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		var replyTo sql.NullString
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Content, &replyTo, &msg.IsEdited, &msg.CreatedAt); err != nil {
			return nil, err
		}
		if replyTo.Valid {
			msg.ReplyToID = replyTo.String
		}
		list = append(list, msg)
	}
	return list, nil
}

func (r *PostgresRepository) UpdateMessageContent(ctx context.Context, msgID, content string) error {
	query := `UPDATE messages SET content = $1, is_edited = true, created_at = created_at WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, content, msgID)
	return err
}

func (r *PostgresRepository) DeleteMessage(ctx context.Context, msgID string) error {
	query := `DELETE FROM messages WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, msgID)
	return err
}

func (r *PostgresRepository) AddReaction(ctx context.Context, rx *domain.MessageReaction) error {
	query := `
		INSERT INTO message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, rx.MessageID, rx.UserID, rx.Emoji).
		Scan(&rx.ID, &rx.CreatedAt)
}

func (r *PostgresRepository) RemoveReaction(ctx context.Context, msgID, userID, emoji string) error {
	query := `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`
	_, err := r.db.ExecContext(ctx, query, msgID, userID, emoji)
	return err
}

func (r *PostgresRepository) PinMessage(ctx context.Context, pin *domain.MessagePin) error {
	query := `
		INSERT INTO message_pins (message_id, room_id, pinned_by)
		VALUES ($1, $2, $3)
		RETURNING pinned_at
	`
	return r.db.QueryRowContext(ctx, query, pin.MessageID, pin.RoomID, pin.PinnedBy).Scan(&pin.PinnedAt)
}

func (r *PostgresRepository) UnpinMessage(ctx context.Context, msgID string) error {
	query := `DELETE FROM message_pins WHERE message_id = $1`
	_, err := r.db.ExecContext(ctx, query, msgID)
	return err
}

func (r *PostgresRepository) GetPinnedCount(ctx context.Context, roomID string) (int, error) {
	query := `SELECT COUNT(*) FROM message_pins WHERE room_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	return count, err
}

func (r *PostgresRepository) SearchMessages(ctx context.Context, roomID, query string) ([]*domain.Message, error) {
	dbQuery := `
		SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at 
		FROM messages 
		WHERE room_id = $1 AND content ILIKE $2
		ORDER BY created_at DESC 
		LIMIT 50
	`
	
	escapedQuery := strings.ReplaceAll(query, "\\", "\\\\")
	escapedQuery = strings.ReplaceAll(escapedQuery, "%", "\\%")
	escapedQuery = strings.ReplaceAll(escapedQuery, "_", "\\_")

	rows, err := r.db.QueryContext(ctx, dbQuery, roomID, "%"+escapedQuery+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*domain.Message{}
	for rows.Next() {
		msg := &domain.Message{}
		var replyTo sql.NullString
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID, &msg.Content, &replyTo, &msg.IsEdited, &msg.CreatedAt); err != nil {
			return nil, err
		}
		if replyTo.Valid {
			msg.ReplyToID = replyTo.String
		}
		list = append(list, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

