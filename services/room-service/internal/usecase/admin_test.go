package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/worktogether/services/room-service/internal/repository"
)

func TestAdminRemoveMemberRequiresAdmin(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	uc := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	if err := uc.AdminRemoveMember(context.Background(), false, "admin-1", "room-1", "member-1"); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAdminRemoveMemberAuditsRemoval(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	uc := NewRoomUsecase(repository.NewPostgresRepository(db), nil)
	mock.ExpectQuery("SELECT id, room_id, user_id, role_id, role_type, active_sub_room_id, muted_until, joined_at FROM room_members").
		WithArgs("room-1", "member-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "room_id", "user_id", "role_id", "role_type", "active_sub_room_id", "muted_until", "joined_at"}).
			AddRow("membership-1", "room-1", "member-1", nil, "MEMBER", nil, nil, time.Now()))
	mock.ExpectExec("DELETE FROM room_members").WithArgs("room-1", "member-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO admin_audit_events").WithArgs("admin-1", "room.member_removed", "room_member", "member-1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	if err := uc.AdminRemoveMember(context.Background(), true, "admin-1", "room-1", "member-1"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
