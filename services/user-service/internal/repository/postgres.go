package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/worktogether/services/user-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateProfile(ctx context.Context, p *domain.UserProfile) error {
	query := `
		INSERT INTO profiles (id, display_name, avatar_url, bio, custom_status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, p.ID, p.DisplayName, p.AvatarURL, p.Bio, p.CustomStatus).
		Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (r *PostgresRepository) GetProfileByID(ctx context.Context, id string) (*domain.UserProfile, error) {
	query := `SELECT id, display_name, avatar_url, bio, custom_status, created_at, updated_at FROM profiles WHERE id = $1`
	p := &domain.UserProfile{}
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&p.ID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CustomStatus, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) UpdateProfile(ctx context.Context, p *domain.UserProfile) error {
	query := `
		UPDATE profiles 
		SET display_name = $1, bio = $2, avatar_url = $3, custom_status = $4, updated_at = NOW()
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query, p.DisplayName, p.Bio, p.AvatarURL, p.CustomStatus, p.ID)
	return err
}

func (r *PostgresRepository) UpdateCustomStatus(ctx context.Context, userID, text string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE profiles SET custom_status = $1, updated_at = NOW() WHERE id = $2`, text, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) AreAcceptedFriends(ctx context.Context, firstUserID, secondUserID string) (bool, error) {
	var accepted bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM friendships
			WHERE status = 'ACCEPTED'
			  AND ((user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1))
		)
	`, firstUserID, secondUserID).Scan(&accepted)
	return accepted, err
}

func (r *PostgresRepository) CreateFriendRequest(ctx context.Context, userID, friendID string) (*domain.Friendship, error) {
	// Kiểm tra xem mối quan hệ đã tồn tại chưa
	queryCheck := `SELECT id, user_id, friend_id, status FROM friendships WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`
	var existing domain.Friendship
	err := r.db.QueryRowContext(ctx, queryCheck, userID, friendID).Scan(&existing.ID, &existing.UserID, &existing.FriendID, &existing.Status)
	if err == nil {
		return &existing, nil // Đã tồn tại quan hệ trước đó
	}

	queryInsert := `
		INSERT INTO friendships (user_id, friend_id, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id, status, created_at, updated_at
	`
	f := &domain.Friendship{
		UserID:   userID,
		FriendID: friendID,
	}
	err = r.db.QueryRowContext(ctx, queryInsert, userID, friendID).Scan(&f.ID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *PostgresRepository) UpdateFriendshipStatus(ctx context.Context, friendshipID, status string) error {
	query := `UPDATE friendships SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, friendshipID)
	return err
}

func (r *PostgresRepository) GetFriendships(ctx context.Context, userID, status string) ([]*domain.Friendship, error) {
	query := `
		SELECT id, user_id, friend_id, status, created_at, updated_at 
		FROM friendships 
		WHERE (user_id = $1 OR friend_id = $1) AND status = $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Friendship
	for rows.Next() {
		f := &domain.Friendship{}
		if err := rows.Scan(&f.ID, &f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func (r *PostgresRepository) BlockUser(ctx context.Context, userID, targetID string) error {
	// Xóa mối quan hệ bạn bè cũ nếu có và thêm dòng chặn mới
	queryDelete := `DELETE FROM friendships WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`
	_, _ = r.db.ExecContext(ctx, queryDelete, userID, targetID)

	queryInsert := `
		INSERT INTO friendships (user_id, friend_id, status)
		VALUES ($1, $2, 'BLOCKED')
	`
	_, err := r.db.ExecContext(ctx, queryInsert, userID, targetID)
	return err
}

// CancelFriendRequest deletes a PENDING friendship where userID is the sender.
func (r *PostgresRepository) CancelFriendRequest(ctx context.Context, friendshipID, userID string) error {
	query := `DELETE FROM friendships WHERE id = $1 AND user_id = $2 AND status = 'PENDING'`
	res, err := r.db.ExecContext(ctx, query, friendshipID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Unfriend deletes an ACCEPTED friendship involving userID.
func (r *PostgresRepository) Unfriend(ctx context.Context, friendshipID, userID string) error {
	query := `DELETE FROM friendships WHERE id = $1 AND (user_id = $2 OR friend_id = $2) AND status = 'ACCEPTED'`
	res, err := r.db.ExecContext(ctx, query, friendshipID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UnblockUser deletes a BLOCKED friendship where userID is the blocker.
func (r *PostgresRepository) UnblockUser(ctx context.Context, friendshipID, userID string) error {
	query := `DELETE FROM friendships WHERE id = $1 AND user_id = $2 AND status = 'BLOCKED'`
	res, err := r.db.ExecContext(ctx, query, friendshipID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
