package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
