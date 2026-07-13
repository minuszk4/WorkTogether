package usecase_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/music-service/internal/repository"
	"github.com/worktogether/services/music-service/internal/usecase"
)

func TestDeleteBookmarkRejectsAnotherUsersBookmark(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	uc := usecase.NewMusicUsecase(repository.NewPostgresRepository(db), "", "", "", "", "")
	rows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "track_id", "position_ms", "note", "created_at"}).
		AddRow("bookmark-1", "room-1", "owner-1", "track-1", 1000, "note", time.Now())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, user_id, track_id, position_ms, note, created_at FROM bookmarks WHERE id = $1")).
		WithArgs("bookmark-1").
		WillReturnRows(rows)

	err = uc.DeleteBookmark(context.Background(), "other-user", "room-1", "bookmark-1")
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database calls: %v", err)
	}
}
