package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/room-service/internal/repository"
)

func TestStartSessionAllowsHostAndCreatesTimeline(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	uc := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	memberRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
		AddRow("member-1", "room-1", "owner-1", nil, "OWNER", nil, nil, time.Now())
	mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
		WithArgs("room-1", "owner-1").
		WillReturnRows(memberRows)
	mock.ExpectQuery("SELECT id, room_id, created_by, title, goal, template_key, status, started_at, ended_at, created_at, updated_at FROM room_sessions").
		WithArgs("room-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "room_id", "created_by", "title", "goal", "template_key", "status", "started_at", "ended_at", "created_at", "updated_at"}))
	mock.ExpectQuery("INSERT INTO room_sessions").
		WithArgs("room-1", "owner-1", "Focus sprint", "Finish the API", "focus").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "started_at", "created_at", "updated_at"}).
			AddRow("session-1", "ACTIVE", time.Now(), time.Now(), time.Now()))
	mock.ExpectQuery("INSERT INTO room_session_timeline").
		WithArgs("session-1", "owner-1", "session.started", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("event-1", time.Now()))
	mock.ExpectExec("UPDATE rooms SET mode = \\$1, updated_at = NOW\\(\\) WHERE id = \\$2").
		WithArgs("focus", "room-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	session, err := uc.StartSession(context.Background(), "owner-1", "room-1", &StartSessionInput{
		Title: "Focus sprint", Goal: "Finish the API", TemplateKey: "focus",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != "session-1" || session.Status != "ACTIVE" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartSessionRejectsMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	uc := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	memberRows := sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
		AddRow("member-1", "room-1", "member-1", nil, "MEMBER", nil, nil, time.Now())
	mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
		WithArgs("room-1", "member-1").
		WillReturnRows(memberRows)

	if _, err := uc.StartSession(context.Background(), "member-1", "room-1", &StartSessionInput{Title: "Nope"}); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetMySessionRecapsReturnsOnlyMemberSessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	uc := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	now := time.Now()
	mock.ExpectQuery("FROM room_sessions s").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "room_id", "room_name", "title", "goal", "template_key", "status", "started_at", "ended_at", "agenda_total", "agenda_done", "actions_total", "actions_done",
		}).AddRow("session-1", "room-1", "Focus Lab", "Ship recap", "Finish dashboard", "focus", "COMPLETED", now, now, 3, 2, 2, 1))

	recaps, err := uc.GetMySessionRecaps(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(recaps) != 1 || recaps[0].RoomName != "Focus Lab" || recaps[0].AgendaDone != 2 || recaps[0].ActionsDone != 1 {
		t.Fatalf("unexpected recaps: %#v", recaps)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
