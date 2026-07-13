package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/room-service/internal/repository"
)

func TestGenerateInviteCode(t *testing.T) {
	u := &RoomUsecase{}

	// Test length
	code := u.generateInviteCode()
	if len(code) != 6 {
		t.Errorf("Expected invite code length to be 6, got %d", len(code))
	}

	// Test charset
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range code {
		if !strings.ContainsRune(charset, char) {
			t.Errorf("Invite code contains invalid character: %c", char)
		}
	}

	// Test randomness/uniqueness (over 100 iterations)
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		c := u.generateInviteCode()
		if codes[c] {
			t.Errorf("Duplicate invite code generated: %s", c)
		}
		codes[c] = true
	}
}

func TestDeleteRoom(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	u := NewRoomUsecase(repo, nil)
	ctx := context.Background()

	t.Run("Success as Owner", func(t *testing.T) {
		userID := "owner-123"
		roomID := "room-123"

		// GetMember mock
		memberRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, userID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, userID).
			WillReturnRows(memberRows)

		// DeleteRoom mock
		mock.ExpectExec("DELETE FROM rooms WHERE id = \\$1").
			WithArgs(roomID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := u.DeleteRoom(ctx, userID, roomID)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Unauthorized as Member", func(t *testing.T) {
		userID := "member-123"
		roomID := "room-123"

		// GetMember mock
		memberRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-2", roomID, userID, "", "MEMBER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, userID).
			WillReturnRows(memberRows)

		err := u.DeleteRoom(ctx, userID, roomID)
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("Expected ErrUnauthorized, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})
}

func TestUpdateRoomMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	u := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	ctx := context.Background()

	memberRows := func(roomID, userID, role string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("member-1", roomID, userID, nil, role, nil, nil, time.Now())
	}

	t.Run("allows moderator to change mode", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs("room-1", "moderator-1").
			WillReturnRows(memberRows("room-1", "moderator-1", "MODERATOR"))
		mock.ExpectExec("UPDATE rooms SET mode = \\$1, updated_at = NOW\\(\\) WHERE id = \\$2").
			WithArgs("focus", "room-1").
			WillReturnResult(sqlmock.NewResult(1, 1))

		mode, err := u.UpdateRoomMode(ctx, "moderator-1", "room-1", "focus")
		if err != nil || mode != "focus" {
			t.Fatalf("got mode=%q err=%v", mode, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects members and invalid modes", func(t *testing.T) {
		if _, err := u.UpdateRoomMode(ctx, "member-1", "room-1", "party"); !errors.Is(err, ErrInvalidRoomMode) {
			t.Fatalf("expected ErrInvalidRoomMode, got %v", err)
		}
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs("room-1", "member-1").
			WillReturnRows(memberRows("room-1", "member-1", "MEMBER"))
		if _, err := u.UpdateRoomMode(ctx, "member-1", "room-1", "chill"); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMuteMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	u := NewRoomUsecase(repo, nil)
	ctx := context.Background()

	t.Run("Success Mute", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"

		// Requester GetMember mock (OWNER)
		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		// Target GetMember mock (MEMBER)
		targetRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-2", roomID, targetID, "", "MEMBER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnRows(targetRows)

		// UpdateMemberMute mock
		mock.ExpectExec("UPDATE room_members SET muted_until = \\$1").
			WithArgs(sqlmock.AnyArg(), roomID, targetID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := u.MuteMember(ctx, requesterID, roomID, targetID, 60)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Cannot Mute Owner", func(t *testing.T) {
		requesterID := "moderator-123"
		targetID := "owner-123"
		roomID := "room-123"

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "MODERATOR", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		targetRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-2", roomID, targetID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnRows(targetRows)

		err := u.MuteMember(ctx, requesterID, roomID, targetID, 60)
		if !errors.Is(err, ErrCannotMuteOwner) {
			t.Errorf("Expected ErrCannotMuteOwner, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Target Member Not Found", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}))

		err := u.MuteMember(ctx, requesterID, roomID, targetID, 60)
		if !errors.Is(err, ErrTargetMemberNotFound) {
			t.Errorf("Expected ErrTargetMemberNotFound, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Database error on requester check", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"
		dbErr := errors.New("db connection failed")

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnError(dbErr)

		err := u.MuteMember(ctx, requesterID, roomID, targetID, 60)
		if !errors.Is(err, dbErr) {
			t.Errorf("Expected dbErr, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Database error on target check", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"
		dbErr := errors.New("db connection failed")

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnError(dbErr)

		err := u.MuteMember(ctx, requesterID, roomID, targetID, 60)
		if !errors.Is(err, dbErr) {
			t.Errorf("Expected dbErr, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})
}

func TestUnmuteMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	u := NewRoomUsecase(repo, nil)
	ctx := context.Background()

	t.Run("Success Unmute", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		targetRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-2", roomID, targetID, "", "MEMBER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnRows(targetRows)

		// UpdateMemberMute mock (nil for unmute)
		mock.ExpectExec("UPDATE room_members SET muted_until = \\$1").
			WithArgs(nil, roomID, targetID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := u.UnmuteMember(ctx, requesterID, roomID, targetID)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Target Member Not Found", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}))

		err := u.UnmuteMember(ctx, requesterID, roomID, targetID)
		if !errors.Is(err, ErrTargetMemberNotFound) {
			t.Errorf("Expected ErrTargetMemberNotFound, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Database error on requester check", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"
		dbErr := errors.New("db connection failed")

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnError(dbErr)

		err := u.UnmuteMember(ctx, requesterID, roomID, targetID)
		if !errors.Is(err, dbErr) {
			t.Errorf("Expected dbErr, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})

	t.Run("Database error on target check", func(t *testing.T) {
		requesterID := "owner-123"
		targetID := "member-123"
		roomID := "room-123"
		dbErr := errors.New("db connection failed")

		reqRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("mem-1", roomID, requesterID, "", "OWNER", nil, nil, time.Now())
		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, requesterID).
			WillReturnRows(reqRows)

		mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
			WithArgs(roomID, targetID).
			WillReturnError(dbErr)

		err := u.UnmuteMember(ctx, requesterID, roomID, targetID)
		if !errors.Is(err, dbErr) {
			t.Errorf("Expected dbErr, got: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Expectations were not met: %s", err)
		}
	})
}
