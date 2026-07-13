package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/playlist-service/internal/repository"
)

func TestGetAllPlaylists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	repo := repository.NewPostgresRepository(db)
	rows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "name", "created_at"}).
		AddRow("playlist-1", "room-1", nil, "Focus", time.Now())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, user_id, name, created_at FROM playlists ORDER BY created_at DESC LIMIT $1")).
		WithArgs(200).
		WillReturnRows(rows)

	playlists, err := repo.GetAllPlaylists(context.Background(), 200)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(playlists) != 1 || playlists[0].ID != "playlist-1" {
		t.Fatalf("unexpected playlists: %+v", playlists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
