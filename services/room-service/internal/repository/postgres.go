package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/worktogether/services/room-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateRoom(ctx context.Context, rm *domain.Room) error {
	if rm.AddMusicPolicy == "" {
		rm.AddMusicPolicy = "all"
	}
	if rm.Theme == "" {
		rm.Theme = "cool-ocean"
	}
	if rm.Mode == "" {
		rm.Mode = "chill"
	}
	query := `
		INSERT INTO rooms (name, description, privacy, password_hash, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode)
		VALUES ($1, $2, UPPER($3), $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, add_music_policy, theme, mode, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, rm.Name, rm.Description, rm.Privacy, rm.PasswordHash, rm.InviteCode, rm.OwnerID, rm.AddMusicPolicy, rm.ParentID, rm.AvatarURL, rm.Rules, rm.Theme, rm.Mode).
		Scan(&rm.ID, &rm.AddMusicPolicy, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt)
}

func (r *PostgresRepository) GetRoomByID(ctx context.Context, id string) (*domain.Room, error) {
	query := `SELECT id, name, description, privacy, password_hash, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode, created_at, updated_at FROM rooms WHERE id = $1`
	rm := &domain.Room{}
	var parentID sql.NullString
	var avatarURL sql.NullString
	var rules sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.PasswordHash, &rm.InviteCode, &rm.OwnerID, &rm.AddMusicPolicy, &parentID, &avatarURL, &rules, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
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

func (r *PostgresRepository) GetRoomByInviteCode(ctx context.Context, code string) (*domain.Room, error) {
	query := `SELECT id, name, description, privacy, password_hash, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode, created_at, updated_at FROM rooms WHERE invite_code = $1`
	rm := &domain.Room{}
	var parentID sql.NullString
	var avatarURL sql.NullString
	var rules sql.NullString
	err := r.db.QueryRowContext(ctx, query, code).
		Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.PasswordHash, &rm.InviteCode, &rm.OwnerID, &rm.AddMusicPolicy, &parentID, &avatarURL, &rules, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
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

func (r *PostgresRepository) GetRooms(ctx context.Context, search string, limit, offset int) ([]*domain.Room, error) {
	query := `
		SELECT id, name, description, privacy, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode, created_at, updated_at
		FROM rooms 
		WHERE privacy = 'PUBLIC' AND (name ILIKE $1 OR description ILIKE $1) AND parent_id IS NULL
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, "%"+search+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Room
	for rows.Next() {
		rm := &domain.Room{}
		var parentID sql.NullString
		var avatarURL sql.NullString
		var rules sql.NullString
		if err := rows.Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.InviteCode, &rm.OwnerID, &rm.AddMusicPolicy, &parentID, &avatarURL, &rules, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt); err != nil {
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
		list = append(list, rm)
	}
	return list, nil
}

func (r *PostgresRepository) UpdateRoom(ctx context.Context, rm *domain.Room) error {
	if rm.AddMusicPolicy == "" {
		rm.AddMusicPolicy = "all"
	}
	if rm.Theme == "" {
		rm.Theme = "cool-ocean"
	}
	query := `
		UPDATE rooms 
		SET name = $1, description = $2, privacy = UPPER($3), password_hash = $4, add_music_policy = $5, avatar_url = $6, rules = $7, theme = $8, updated_at = NOW() 
		WHERE id = $9
	`
	_, err := r.db.ExecContext(ctx, query, rm.Name, rm.Description, rm.Privacy, rm.PasswordHash, rm.AddMusicPolicy, rm.AvatarURL, rm.Rules, rm.Theme, rm.ID)
	return err
}

func (r *PostgresRepository) UpdateRoomMode(ctx context.Context, roomID, mode string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE rooms SET mode = $1, updated_at = NOW() WHERE id = $2`, mode, roomID)
	return err
}

func (r *PostgresRepository) DeleteRoom(ctx context.Context, id string) error {
	query := `DELETE FROM rooms WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) AddMember(ctx context.Context, m *domain.RoomMember) error {
	query := `
		INSERT INTO room_members (room_id, user_id, role_type, role_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, joined_at
	`
	var roleID interface{} = m.RoleID
	if m.RoleID == "" {
		roleID = nil
	}
	return r.db.QueryRowContext(ctx, query, m.RoomID, m.UserID, m.RoleType, roleID).
		Scan(&m.ID, &m.JoinedAt)
}

func (r *PostgresRepository) GetMember(ctx context.Context, roomID, userID string) (*domain.RoomMember, error) {
	query := `SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members WHERE room_id = $1 AND user_id = $2`
	m := &domain.RoomMember{}
	var roleID sql.NullString
	var activeSubRoomID sql.NullString
	var mutedUntil sql.NullTime
	err := r.db.QueryRowContext(ctx, query, roomID, userID).
		Scan(&m.ID, &m.RoomID, &m.UserID, &roleID, &m.RoleType, &activeSubRoomID, &mutedUntil, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if roleID.Valid {
		m.RoleID = roleID.String
	}
	if activeSubRoomID.Valid {
		m.ActiveSubRoomID = &activeSubRoomID.String
	}
	if mutedUntil.Valid {
		m.MutedUntil = &mutedUntil.Time
	}
	return m, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, roomID string) ([]*domain.RoomMember, error) {
	query := `
		SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at
		FROM room_members
		WHERE room_id = $1
		ORDER BY
			CASE role_type
				WHEN 'OWNER' THEN 0
				WHEN 'MODERATOR' THEN 1
				ELSE 2
			END,
			joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.RoomMember
	for rows.Next() {
		member := &domain.RoomMember{}
		var roleID sql.NullString
		var activeSubRoomID sql.NullString
		var mutedUntil sql.NullTime
		if err := rows.Scan(&member.ID, &member.RoomID, &member.UserID, &roleID, &member.RoleType, &activeSubRoomID, &mutedUntil, &member.JoinedAt); err != nil {
			return nil, err
		}
		if roleID.Valid {
			member.RoleID = roleID.String
		}
		if activeSubRoomID.Valid {
			member.ActiveSubRoomID = &activeSubRoomID.String
		}
		if mutedUntil.Valid {
			member.MutedUntil = &mutedUntil.Time
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	query := `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *PostgresRepository) UpdateMemberRole(ctx context.Context, roomID, userID, roleID, roleType string) error {
	query := `UPDATE room_members SET role_id = $1, role_type = $2 WHERE room_id = $3 AND user_id = $4`
	var rID interface{} = roleID
	if roleID == "" {
		rID = nil
	}
	_, err := r.db.ExecContext(ctx, query, rID, roleType, roomID, userID)
	return err
}

func (r *PostgresRepository) CreateRole(ctx context.Context, rl *domain.RoomRole) error {
	query := `
		INSERT INTO room_roles (room_id, name, can_chat, can_manage_playlist, can_control_playback, can_moderate_members, can_use_voice)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, rl.RoomID, rl.Name, rl.CanChat, rl.CanManagePlaylist, rl.CanControlPlayback, rl.CanModerateMembers, rl.CanUseVoice).
		Scan(&rl.ID, &rl.CreatedAt, &rl.UpdatedAt)
}

func (r *PostgresRepository) GetRoleByID(ctx context.Context, id string) (*domain.RoomRole, error) {
	query := `SELECT id, room_id, name, can_chat, can_manage_playlist, can_control_playback, can_moderate_members, can_use_voice, created_at, updated_at FROM room_roles WHERE id = $1`
	rl := &domain.RoomRole{}
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&rl.ID, &rl.RoomID, &rl.Name, &rl.CanChat, &rl.CanManagePlaylist, &rl.CanControlPlayback, &rl.CanModerateMembers, &rl.CanUseVoice, &rl.CreatedAt, &rl.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rl, nil
}

func (r *PostgresRepository) GetRolesByRoom(ctx context.Context, roomID string) ([]*domain.RoomRole, error) {
	query := `
		SELECT id, room_id, name, can_chat, can_manage_playlist, can_control_playback, can_moderate_members, can_use_voice, created_at, updated_at 
		FROM room_roles 
		WHERE room_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.RoomRole
	for rows.Next() {
		rl := &domain.RoomRole{}
		if err := rows.Scan(&rl.ID, &rl.RoomID, &rl.Name, &rl.CanChat, &rl.CanManagePlaylist, &rl.CanControlPlayback, &rl.CanModerateMembers, &rl.CanUseVoice, &rl.CreatedAt, &rl.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, rl)
	}
	return list, nil
}

func (r *PostgresRepository) AddBan(ctx context.Context, b *domain.RoomBan) error {
	query := `
		INSERT INTO room_bans (room_id, user_id, banned_by, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, b.RoomID, b.UserID, b.BannedBy, b.Reason).
		Scan(&b.ID, &b.CreatedAt)
}

func (r *PostgresRepository) IsBanned(ctx context.Context, roomID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM room_bans WHERE room_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, roomID, userID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) RemoveBan(ctx context.Context, roomID, userID string) error {
	query := `DELETE FROM room_bans WHERE room_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *PostgresRepository) GetSubRooms(ctx context.Context, parentID string) ([]*domain.Room, error) {
	query := `SELECT id, name, description, privacy, invite_code, owner_id, add_music_policy, parent_id, avatar_url, rules, theme, mode, created_at, updated_at FROM rooms WHERE parent_id = $1`
	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Room
	for rows.Next() {
		rm := &domain.Room{}
		var parentIDStr sql.NullString
		var avatarURL sql.NullString
		var rules sql.NullString
		if err := rows.Scan(&rm.ID, &rm.Name, &rm.Description, &rm.Privacy, &rm.InviteCode, &rm.OwnerID, &rm.AddMusicPolicy, &parentIDStr, &avatarURL, &rules, &rm.Theme, &rm.Mode, &rm.CreatedAt, &rm.UpdatedAt); err != nil {
			return nil, err
		}
		if parentIDStr.Valid {
			rm.ParentID = &parentIDStr.String
		}
		if avatarURL.Valid {
			rm.AvatarURL = &avatarURL.String
		}
		if rules.Valid {
			rm.Rules = &rules.String
		}
		list = append(list, rm)
	}
	return list, nil
}

func (r *PostgresRepository) MoveMember(ctx context.Context, roomID string, userID string, subRoomID *string) error {
	query := `UPDATE room_members SET active_sub_room_id = $1 WHERE room_id = $2 AND user_id = $3`
	var subRoomVal interface{} = subRoomID
	if subRoomID != nil && *subRoomID == "" {
		subRoomVal = nil
	}
	_, err := r.db.ExecContext(ctx, query, subRoomVal, roomID, userID)
	return err
}

func (r *PostgresRepository) UpdateMemberMute(ctx context.Context, roomID, userID string, mutedUntil *time.Time) error {
	query := `UPDATE room_members SET muted_until = $1 WHERE room_id = $2 AND user_id = $3`
	_, err := r.db.ExecContext(ctx, query, mutedUntil, roomID, userID)
	return err
}
