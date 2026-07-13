package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/worktogether/services/room-service/internal/domain"
)

type AdminAuditEvent struct {
	ID         string `json:"id"`
	ActorID    string `json:"actor_id"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	CreatedAt  string `json:"created_at"`
}

func scanAdminRoom(rows *sql.Rows) (*domain.Room, error) {
	rm := &domain.Room{}
	var parentID, avatarURL, rules sql.NullString
	err := rows.Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.PasswordHash, &rm.InviteCode, &rm.OwnerID, &rm.AddMusicPolicy, &parentID, &avatarURL, &rules, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		rm.ParentID = &parentID.String
	}
	if avatarURL.Valid {
		rm.AvatarURL = &avatarURL.String
	}
	if rules.Valid {
		rm.Rules = &rules.String
	}
	return rm, nil
}

func (r *PostgresRepository) ListAdminRooms(ctx context.Context, limit int) ([]*domain.Room, error) {
	query := `SELECT id, name, description, privacy, password_hash, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode, created_at, updated_at FROM rooms ORDER BY created_at DESC LIMIT $1`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rooms := []*domain.Room{}
	for rows.Next() {
		rm, err := scanAdminRoom(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, rm)
	}
	return rooms, rows.Err()
}

func (r *PostgresRepository) CreateAdminAuditEvent(ctx context.Context, actorID, action, targetType, targetID string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO admin_audit_events (actor_id, action, target_type, target_id, payload) VALUES ($1, $2, $3, $4, $5)`, actorID, action, targetType, targetID, data)
	return err
}

func (r *PostgresRepository) ListAdminAuditEvents(ctx context.Context, limit int) ([]*AdminAuditEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, actor_id, action, target_type, target_id, created_at FROM admin_audit_events ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []*AdminAuditEvent{}
	for rows.Next() {
		event := &AdminAuditEvent{}
		if err := rows.Scan(&event.ID, &event.ActorID, &event.Action, &event.TargetType, &event.TargetID, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
