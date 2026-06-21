package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/music-service/internal/domain"
)

func TestPostgresRepository_Lyrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := &PostgresRepository{db: db}
	ctx := context.Background()

	t.Run("SaveLyrics", func(t *testing.T) {
		trackID := "track-123"
		content := "[00:10.00]Hello world"

		mock.ExpectExec("INSERT INTO track_lyrics").
			WithArgs(trackID, content).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.SaveLyrics(ctx, trackID, content)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("GetLyrics - Success", func(t *testing.T) {
		trackID := "track-123"
		expectedContent := "[00:10.00]Hello world"

		rows := sqlmock.NewRows([]string{"content"}).AddRow(expectedContent)
		mock.ExpectQuery("SELECT content FROM track_lyrics").
			WithArgs(trackID).
			WillReturnRows(rows)

		content, err := repo.GetLyrics(ctx, trackID)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if content != expectedContent {
			t.Errorf("Expected content %q, got %q", expectedContent, content)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("GetLyrics - Not Found", func(t *testing.T) {
		trackID := "track-not-exist"

		mock.ExpectQuery("SELECT content FROM track_lyrics").
			WithArgs(trackID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetLyrics(ctx, trackID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})
}

func TestPostgresRepository_Bookmarks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := &PostgresRepository{db: db}
	ctx := context.Background()

	t.Run("SaveBookmark", func(t *testing.T) {
		b := &domain.Bookmark{
			ID:         "bookmark-1",
			RoomID:     "room-123",
			UserID:     "user-456",
			TrackID:    "track-789",
			PositionMS: 45000,
			Note:       "Great drops!",
		}

		mock.ExpectExec("INSERT INTO bookmarks").
			WithArgs(b.ID, b.RoomID, b.UserID, b.TrackID, b.PositionMS, b.Note).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.SaveBookmark(ctx, b)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("GetBookmarks", func(t *testing.T) {
		roomID := "room-123"
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "track_id", "position_ms", "note", "created_at"}).
			AddRow("bookmark-1", roomID, "user-456", "track-789", 45000, "Great drops!", now).
			AddRow("bookmark-2", roomID, "user-789", "track-789", 90000, "Incredible solo", now)

		mock.ExpectQuery("SELECT id, room_id, user_id, track_id, position_ms, note, created_at FROM bookmarks").
			WithArgs(roomID).
			WillReturnRows(rows)

		list, err := repo.GetBookmarks(ctx, roomID)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if len(list) != 2 {
			t.Errorf("Expected 2 bookmarks, got %d", len(list))
		}
		if list[0].ID != "bookmark-1" || list[1].ID != "bookmark-2" {
			t.Errorf("Returned bookmarks mismatch")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("DeleteBookmark", func(t *testing.T) {
		bookmarkID := "bookmark-1"

		mock.ExpectExec("DELETE FROM bookmarks").
			WithArgs(bookmarkID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteBookmark(ctx, bookmarkID)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})
}
