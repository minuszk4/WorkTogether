package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/worktogether/services/room-service/internal/domain"
)

func (r *PostgresRepository) CreateSession(ctx context.Context, session *domain.RoomSession) error {
	query := `
		INSERT INTO room_sessions (room_id, created_by, title, goal, template_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, started_at, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, session.RoomID, session.CreatedBy, session.Title, session.Goal, session.TemplateKey).
		Scan(&session.ID, &session.Status, &session.StartedAt, &session.CreatedAt, &session.UpdatedAt)
}

func (r *PostgresRepository) GetActiveSession(ctx context.Context, roomID string) (*domain.RoomSession, error) {
	query := `SELECT id, room_id, created_by, title, goal, template_key, status, started_at, ended_at, created_at, updated_at FROM room_sessions WHERE room_id = $1 AND status = 'ACTIVE' ORDER BY started_at DESC LIMIT 1`
	return r.scanSession(r.db.QueryRowContext(ctx, query, roomID))
}

func (r *PostgresRepository) GetSession(ctx context.Context, roomID, sessionID string) (*domain.RoomSession, error) {
	query := `SELECT id, room_id, created_by, title, goal, template_key, status, started_at, ended_at, created_at, updated_at FROM room_sessions WHERE room_id = $1 AND id = $2`
	return r.scanSession(r.db.QueryRowContext(ctx, query, roomID, sessionID))
}

func (r *PostgresRepository) CompleteSession(ctx context.Context, roomID, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE room_sessions SET status = 'COMPLETED', ended_at = NOW(), updated_at = NOW() WHERE id = $1 AND room_id = $2 AND status = 'ACTIVE'`, sessionID, roomID)
	return err
}

func (r *PostgresRepository) CreateAgendaItem(ctx context.Context, item *domain.SessionAgendaItem) error {
	query := `INSERT INTO room_session_agenda (session_id, content, position) VALUES ($1, $2, $3) RETURNING id, is_done, created_at`
	return r.db.QueryRowContext(ctx, query, item.SessionID, item.Content, item.Position).
		Scan(&item.ID, &item.IsDone, &item.CreatedAt)
}

func (r *PostgresRepository) UpdateAgendaItem(ctx context.Context, sessionID, itemID string, isDone bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE room_session_agenda SET is_done = $1 WHERE id = $2 AND session_id = $3`, isDone, itemID, sessionID)
	return err
}

func (r *PostgresRepository) ListAgendaItems(ctx context.Context, sessionID string) ([]*domain.SessionAgendaItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, session_id, content, position, is_done, created_at FROM room_session_agenda WHERE session_id = $1 ORDER BY position, created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*domain.SessionAgendaItem{}
	for rows.Next() {
		item := &domain.SessionAgendaItem{}
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Content, &item.Position, &item.IsDone, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) CreateActionItem(ctx context.Context, item *domain.SessionActionItem) error {
	query := `INSERT INTO room_session_actions (session_id, content, assignee_id, due_at, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id, status, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, item.SessionID, item.Content, item.AssigneeID, item.DueAt, item.CreatedBy).
		Scan(&item.ID, &item.Status, &item.CreatedAt, &item.UpdatedAt)
}

func (r *PostgresRepository) UpdateActionStatus(ctx context.Context, sessionID, itemID, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE room_session_actions SET status = $1, updated_at = NOW() WHERE id = $2 AND session_id = $3`, status, itemID, sessionID)
	return err
}

func (r *PostgresRepository) ListActionItems(ctx context.Context, sessionID string) ([]*domain.SessionActionItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, session_id, content, assignee_id, due_at, status, created_by, created_at, updated_at FROM room_session_actions WHERE session_id = $1 ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*domain.SessionActionItem{}
	for rows.Next() {
		item := &domain.SessionActionItem{}
		var assigneeID sql.NullString
		var dueAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Content, &assigneeID, &dueAt, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if assigneeID.Valid {
			item.AssigneeID = &assigneeID.String
		}
		if dueAt.Valid {
			item.DueAt = &dueAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) ListPersonalActionItems(ctx context.Context, userID string) ([]*domain.PersonalActionItem, error) {
	query := `
		SELECT a.id, a.session_id, a.content, a.assignee_id, a.due_at, a.status, a.created_by, a.created_at, a.updated_at, r.id, r.name, s.title
		FROM room_session_actions a
		JOIN room_sessions s ON s.id = a.session_id
		JOIN rooms r ON r.id = s.room_id
		WHERE a.assignee_id = $1 OR a.created_by = $1
		ORDER BY CASE WHEN a.status = 'OPEN' THEN 0 ELSE 1 END, a.due_at NULLS LAST, a.updated_at DESC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*domain.PersonalActionItem{}
	for rows.Next() {
		item := &domain.PersonalActionItem{}
		var assigneeID sql.NullString
		var dueAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Content, &assigneeID, &dueAt, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.RoomID, &item.RoomName, &item.SessionTitle); err != nil {
			return nil, err
		}
		if assigneeID.Valid {
			item.AssigneeID = &assigneeID.String
		}
		if dueAt.Valid {
			item.DueAt = &dueAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) CreateSessionTimelineEvent(ctx context.Context, event *domain.SessionTimelineEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	query := `INSERT INTO room_session_timeline (session_id, actor_id, event_type, payload) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, event.SessionID, event.ActorID, event.EventType, payload).
		Scan(&event.ID, &event.CreatedAt)
}

func (r *PostgresRepository) ListSessionTimeline(ctx context.Context, sessionID string) ([]*domain.SessionTimelineEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, session_id, actor_id, event_type, payload, created_at FROM room_session_timeline WHERE session_id = $1 ORDER BY created_at DESC LIMIT 100`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []*domain.SessionTimelineEvent{}
	for rows.Next() {
		event := &domain.SessionTimelineEvent{}
		var payload []byte
		if err := rows.Scan(&event.ID, &event.SessionID, &event.ActorID, &event.EventType, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) scanSession(row *sql.Row) (*domain.RoomSession, error) {
	session := &domain.RoomSession{}
	var endedAt sql.NullTime
	err := row.Scan(&session.ID, &session.RoomID, &session.CreatedBy, &session.Title, &session.Goal, &session.TemplateKey, &session.Status, &session.StartedAt, &endedAt, &session.CreatedAt, &session.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	return session, nil
}
