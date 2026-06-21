package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/chat-service/internal/domain"
	"github.com/worktogether/services/chat-service/internal/repository"
	"github.com/worktogether/services/chat-service/internal/usecase"
)

func TestEditMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewChatUsecase(repo, nil)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-1"
		content := "new content"
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, roomID, userID, "old content", sql.NullString{}, false, createdAt)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(rows)

		mock.ExpectExec(regexp.QuoteMeta("UPDATE messages SET content = $1, is_edited = true, created_at = created_at WHERE id = $2")).
			WithArgs(content, msgID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		msg, err := uc.EditMessage(ctx, userID, roomID, msgID, content)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if msg == nil || msg.Content != content || !msg.IsEdited {
			t.Errorf("unexpected message state: %+v", msg)
		}
	})

	t.Run("Cross-Room Message Tampering Rejected", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-different"
		content := "new content"
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, "room-actual", userID, "old content", sql.NullString{}, false, createdAt)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(rows)

		_, err := uc.EditMessage(ctx, userID, roomID, msgID, content)
		if !errors.Is(err, usecase.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Unauthorized Edit", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-unauthorized"
		roomID := "room-1"
		content := "new content"
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, roomID, "user-owner", "old content", sql.NullString{}, false, createdAt)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(rows)

		_, err := uc.EditMessage(ctx, userID, roomID, msgID, content)
		if !errors.Is(err, usecase.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Edit Time Expired", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-1"
		content := "new content"
		createdAt := time.Now().Add(-20 * time.Minute)

		rows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, roomID, userID, "old content", sql.NullString{}, false, createdAt)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(rows)

		_, err := uc.EditMessage(ctx, userID, roomID, msgID, content)
		if !errors.Is(err, usecase.ErrEditTimeExpired) {
			t.Errorf("expected ErrEditTimeExpired, got %v", err)
		}
	})
}

func TestPinMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewChatUsecase(repo, nil)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-1"

		msgRows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, roomID, userID, "content", sql.NullString{}, false, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(msgRows)

		pinnedCountRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM message_pins WHERE room_id = $1")).
			WithArgs(roomID).
			WillReturnRows(pinnedCountRows)

		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO message_pins (message_id, room_id, pinned_by) VALUES ($1, $2, $3) RETURNING pinned_at")).
			WithArgs(msgID, roomID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"pinned_at"}).AddRow(time.Now()))

		pin, err := uc.PinMessage(ctx, userID, roomID, msgID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if pin == nil || pin.MessageID != msgID || pin.RoomID != roomID || pin.PinnedBy != userID {
			t.Errorf("unexpected pin state: %+v", pin)
		}
	})

	t.Run("Cross-Room Pin Rejected", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-different"

		msgRows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, "room-actual", userID, "content", sql.NullString{}, false, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(msgRows)

		_, err := uc.PinMessage(ctx, userID, roomID, msgID)
		if !errors.Is(err, usecase.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Pin Limit Exceeded", func(t *testing.T) {
		msgID := "msg-1"
		userID := "user-1"
		roomID := "room-1"

		msgRows := sqlmock.NewRows([]string{"id", "room_id", "sender_id", "content", "reply_to_id", "is_edited", "created_at"}).
			AddRow(msgID, roomID, userID, "content", sql.NullString{}, false, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, room_id, sender_id, content, reply_to_id, is_edited, created_at FROM messages WHERE id = $1")).
			WithArgs(msgID).
			WillReturnRows(msgRows)

		pinnedCountRows := sqlmock.NewRows([]string{"count"}).AddRow(10)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM message_pins WHERE room_id = $1")).
			WithArgs(roomID).
			WillReturnRows(pinnedCountRows)

		_, err := uc.PinMessage(ctx, userID, roomID, msgID)
		if err == nil || err.Error() != "vượt quá giới hạn 10 tin nhắn ghim cho phòng này" {
			t.Errorf("expected pin limit error, got %v", err)
		}
	})
}

func TestSaveMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewChatUsecase(repo, nil)

	ctx := context.Background()

	t.Run("Success without Mentions", func(t *testing.T) {
		senderID := "user-1"
		roomID := "room-1"
		req := &domain.SendMessagePayload{
			Content: "hello",
		}

		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO messages (id, room_id, sender_id, content, reply_to_id, is_edited, created_at)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		msg, err := uc.SaveMessage(ctx, senderID, roomID, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if msg == nil || msg.Content != "hello" || msg.SenderID != senderID || msg.RoomID != roomID {
			t.Errorf("unexpected message state: %+v", msg)
		}
	})

	t.Run("Success with Mentions", func(t *testing.T) {
		senderID := "user-1"
		roomID := "room-1"
		req := &domain.SendMessagePayload{
			Content:  "hello @user-2",
			Mentions: []string{"user-2"},
		}

		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO messages (id, room_id, sender_id, content, reply_to_id, is_edited, created_at)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		msg, err := uc.SaveMessage(ctx, senderID, roomID, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if msg == nil || msg.Content != "hello @user-2" || msg.SenderID != senderID || msg.RoomID != roomID {
			t.Errorf("unexpected message state: %+v", msg)
		}
	})

	t.Run("Success with Duplicate and Self Mentions", func(t *testing.T) {
		senderID := "user-1"
		roomID := "room-1"
		req := &domain.SendMessagePayload{
			Content:  "hello @user-2 @user-1",
			Mentions: []string{"user-2", "user-1", "user-2", "", "user-3"},
		}

		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO messages (id, room_id, sender_id, content, reply_to_id, is_edited, created_at)")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		msg, err := uc.SaveMessage(ctx, senderID, roomID, req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if msg == nil || msg.SenderID != senderID || msg.RoomID != roomID {
			t.Errorf("unexpected message state: %+v", msg)
		}
		// Expect Mentions to have only "user-2" and "user-3"
		expectedMentions := []string{"user-2", "user-3"}
		if len(req.Mentions) != len(expectedMentions) {
			t.Errorf("expected mentions %v, got %v", expectedMentions, req.Mentions)
		} else {
			for i, m := range req.Mentions {
				if m != expectedMentions[i] {
					t.Errorf("expected mention at %d to be %s, got %s", i, expectedMentions[i], m)
				}
			}
		}
	})
}

