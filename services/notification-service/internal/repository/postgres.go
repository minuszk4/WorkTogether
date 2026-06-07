package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/worktogether/services/notification-service/internal/domain"
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

	schema := `
	CREATE TABLE IF NOT EXISTS notifications (
		id VARCHAR(36) PRIMARY KEY,
		receiver_id VARCHAR(36) NOT NULL,
		sender_id VARCHAR(36) NOT NULL,
		type VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.ExecContext(ctx, schema); err != nil {
		return err
	}

	log.Println("Đã khởi tạo schema PostgreSQL cho notification-service thành công.")
	return nil
}

func (r *PostgresRepository) SaveNotification(ctx context.Context, n *domain.Notification) error {
	query := `INSERT INTO notifications (id, receiver_id, sender_id, type, content, is_read, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, n.ID, n.ReceiverID, n.SenderID, n.Type, n.Content, n.IsRead, n.CreatedAt)
	return err
}

func (r *PostgresRepository) GetUserNotifications(ctx context.Context, userID string, limit int) ([]*domain.Notification, error) {
	query := `SELECT id, receiver_id, sender_id, type, content, is_read, created_at 
	          FROM notifications 
	          WHERE receiver_id = $1 
	          ORDER BY created_at DESC 
	          LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.ReceiverID, &n.SenderID, &n.Type, &n.Content, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &n)
	}
	return list, nil
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1 AND receiver_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}

func (r *PostgresRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE receiver_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE receiver_id = $1 AND is_read = FALSE`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}
