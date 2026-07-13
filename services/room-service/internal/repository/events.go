package repository

import (
	"context"
	"database/sql"

	"github.com/worktogether/services/room-service/internal/domain"
)

func (r *PostgresRepository) CreateRoomEvent(ctx context.Context, event *domain.RoomEvent) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO room_events (room_id, created_by, title, description, starts_at) VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`, event.RoomID, event.CreatedBy, event.Title, event.Description, event.StartsAt).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
}

func (r *PostgresRepository) ListUpcomingRoomEvents(ctx context.Context, roomID string) ([]*domain.RoomEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, room_id, created_by, title, description, starts_at, reminder_sent_at, cancelled_at, created_at, updated_at FROM room_events WHERE room_id = $1 AND cancelled_at IS NULL AND starts_at >= NOW() ORDER BY starts_at LIMIT 50`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []*domain.RoomEvent{}
	for rows.Next() {
		event, err := scanRoomEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func scanRoomEvent(row interface{ Scan(...any) error }) (*domain.RoomEvent, error) {
	event := &domain.RoomEvent{}
	var reminder, cancelled sql.NullTime
	err := row.Scan(&event.ID, &event.RoomID, &event.CreatedBy, &event.Title, &event.Description, &event.StartsAt, &reminder, &cancelled, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if reminder.Valid {
		event.ReminderSentAt = &reminder.Time
	}
	if cancelled.Valid {
		event.CancelledAt = &cancelled.Time
	}
	return event, nil
}
