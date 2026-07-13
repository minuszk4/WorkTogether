package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/worktogether/services/auth-service/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateAccount(ctx context.Context, acc *domain.Account) error {
	query := `
		INSERT INTO accounts (email, username, password_hash, is_verified, google_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	var googleID *string
	if acc.GoogleID != "" {
		googleID = &acc.GoogleID
	}
	return r.db.QueryRowContext(ctx, query, acc.Email, acc.Username, nullableString(acc.PasswordHash), acc.IsVerified, googleID).
		Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id string) (*domain.Account, error) {
	query := `SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE id = $1`
	acc := &domain.Account{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&acc.ID, &acc.Email, &acc.Username, &acc.PasswordHash, &acc.IsVerified, &acc.IsAdmin, &acc.GoogleID, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return acc, nil
}

func (r *PostgresRepository) GetAccountByIdentity(ctx context.Context, identity string) (*domain.Account, error) {
	query := `SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE email = $1 OR username = $2`
	acc := &domain.Account{}
	err := r.db.QueryRowContext(ctx, query, identity, identity).Scan(
		&acc.ID, &acc.Email, &acc.Username, &acc.PasswordHash, &acc.IsVerified, &acc.IsAdmin, &acc.GoogleID, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return acc, nil
}

// GetAccountByGoogleID tìm tài khoản theo Google ID
func (r *PostgresRepository) GetAccountByGoogleID(ctx context.Context, googleID string) (*domain.Account, error) {
	query := `SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE google_id = $1`
	acc := &domain.Account{}
	err := r.db.QueryRowContext(ctx, query, googleID).Scan(
		&acc.ID, &acc.Email, &acc.Username, &acc.PasswordHash, &acc.IsVerified, &acc.IsAdmin, &acc.GoogleID, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return acc, nil
}

// GetAccountByEmail tìm tài khoản theo email chính xác
func (r *PostgresRepository) GetAccountByEmail(ctx context.Context, email string) (*domain.Account, error) {
	query := `SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE email = $1`
	acc := &domain.Account{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&acc.ID, &acc.Email, &acc.Username, &acc.PasswordHash, &acc.IsVerified, &acc.IsAdmin, &acc.GoogleID, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return acc, nil
}

// LinkGoogleID liên kết google_id vào tài khoản có sẵn (đăng nhập email trước, sau link Google)
func (r *PostgresRepository) LinkGoogleID(ctx context.Context, accountID, googleID string) error {
	query := `UPDATE accounts SET google_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, googleID, accountID)
	return err
}

func (r *PostgresRepository) UpdateAccountVerification(ctx context.Context, id string, isVerified bool) error {
	query := `UPDATE accounts SET is_verified = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, isVerified, id)
	return err
}

func (r *PostgresRepository) ListAccounts(ctx context.Context, limit int) ([]*domain.Account, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(is_suspended,false), COALESCE(google_id,''), created_at, updated_at FROM accounts ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []*domain.Account{}
	for rows.Next() {
		acc := &domain.Account{}
		if err := rows.Scan(&acc.ID, &acc.Email, &acc.Username, &acc.PasswordHash, &acc.IsVerified, &acc.IsAdmin, &acc.IsSuspended, &acc.GoogleID, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, rows.Err()
}

func (r *PostgresRepository) SetAccountAdmin(ctx context.Context, accountID string, isAdmin bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE accounts SET is_admin = $1, updated_at = NOW() WHERE id = $2`, isAdmin, accountID)
	return err
}

func (r *PostgresRepository) IsAccountSuspended(ctx context.Context, accountID string) (bool, error) {
	var suspended bool
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(is_suspended,false) FROM accounts WHERE id = $1`, accountID).Scan(&suspended)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return suspended, err
}

func (r *PostgresRepository) SetAccountSuspended(ctx context.Context, accountID string, suspended bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE accounts SET is_suspended = $1, updated_at = NOW() WHERE id = $2`, suspended, accountID)
	return err
}

func (r *PostgresRepository) CreateSession(ctx context.Context, sess *domain.Session) error {
	query := `
		INSERT INTO sessions (account_id, refresh_token, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, sess.AccountID, sess.RefreshToken, sess.IPAddress, sess.UserAgent, sess.ExpiresAt).
		Scan(&sess.ID, &sess.CreatedAt)
}

func (r *PostgresRepository) GetSessionByToken(ctx context.Context, token string) (*domain.Session, error) {
	query := `SELECT id, account_id, refresh_token, ip_address, user_agent, expires_at, created_at FROM sessions WHERE refresh_token = $1`
	sess := &domain.Session{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&sess.ID, &sess.AccountID, &sess.RefreshToken, &sess.IPAddress, &sess.UserAgent, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return sess, nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *PostgresRepository) DeleteSessionsByAccountID(ctx context.Context, accountID string) error {
	query := `DELETE FROM sessions WHERE account_id = $1`
	_, err := r.db.ExecContext(ctx, query, accountID)
	return err
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	query := `UPDATE accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, passwordHash, id)
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

// nullableString chuyển empty string thành nil (NULL trong DB)
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
