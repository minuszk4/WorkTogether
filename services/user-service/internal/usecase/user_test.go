package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	roomv1 "github.com/worktogether/services/user-service/api/v1"
	"github.com/worktogether/services/user-service/internal/domain"
	"google.golang.org/grpc"
)

type customStatusRepositoryFake struct {
	userID string
	text   string
	err    error
}

type acceptedFriendsFake struct{ accepted bool }

func (r acceptedFriendsFake) AreAcceptedFriends(context.Context, string, string) (bool, error) {
	return r.accepted, nil
}

type roomClientFake struct{ members map[string]bool }

func (r roomClientFake) VerifyRoomMember(_ context.Context, request *roomv1.VerifyRoomMemberRequest, _ ...grpc.CallOption) (*roomv1.VerifyRoomMemberResponse, error) {
	return &roomv1.VerifyRoomMemberResponse{IsMember: r.members[request.UserID]}, nil
}

func (r *customStatusRepositoryFake) UpdateCustomStatus(_ context.Context, userID, text string) error {
	r.userID = userID
	r.text = text
	return r.err
}

type presenceRepositoryFake struct {
	presence *domain.Presence
	set      *domain.Presence
}

func (r *presenceRepositoryFake) GetPresence(context.Context, string) (*domain.Presence, error) {
	return r.presence, nil
}

func (r *presenceRepositoryFake) SetPresence(_ context.Context, _ string, presence *domain.Presence) error {
	r.set = presence
	return nil
}

func TestHeartbeatPresenceOnlyWritesPresence(t *testing.T) {
	presence := &presenceRepositoryFake{}
	uc := &UserUsecase{presenceRepo: presence}

	if _, err := uc.HeartbeatPresence(context.Background(), "user-1", &domain.HeartbeatRequest{Status: "away"}); err != nil {
		t.Fatal(err)
	}
	if presence.set == nil || presence.set.Status != "away" {
		t.Fatalf("saved presence = %#v, want away", presence.set)
	}
	if presence.set.CustomText != "" {
		t.Fatalf("heartbeat custom text = %q, want empty", presence.set.CustomText)
	}
}

func TestUpdateCustomStatusTrimsAndPersistsText(t *testing.T) {
	profiles := &customStatusRepositoryFake{}
	uc := &UserUsecase{
		customStatusRepo: profiles,
		presenceRepo:     &presenceRepositoryFake{presence: &domain.Presence{Status: "online"}},
	}

	presence, err := uc.UpdateCustomStatus(context.Background(), "user-1", "  studying  ")
	if err != nil {
		t.Fatal(err)
	}
	if profiles.userID != "user-1" || profiles.text != "studying" {
		t.Fatalf("saved profile update = (%q, %q), want (%q, %q)", profiles.userID, profiles.text, "user-1", "studying")
	}
	if presence.CustomText != "studying" {
		t.Fatalf("presence custom text = %q, want %q", presence.CustomText, "studying")
	}
}

func TestUpdateCustomStatusClearsWithEmptyText(t *testing.T) {
	profiles := &customStatusRepositoryFake{}
	uc := &UserUsecase{
		customStatusRepo: profiles,
		presenceRepo:     &presenceRepositoryFake{presence: &domain.Presence{Status: "online", CustomText: "old"}},
	}

	_, err := uc.UpdateCustomStatus(context.Background(), "user-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if profiles.text != "" {
		t.Fatalf("saved profile update = %q, want empty text", profiles.text)
	}
}

func TestUpdateCustomStatusRejectsTextOver100Characters(t *testing.T) {
	profiles := &customStatusRepositoryFake{}
	uc := &UserUsecase{
		customStatusRepo: profiles,
		presenceRepo:     &presenceRepositoryFake{presence: &domain.Presence{Status: "online"}},
	}

	_, err := uc.UpdateCustomStatus(context.Background(), "user-1", strings.Repeat("a", 101))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if profiles.userID != "" || profiles.text != "" {
		t.Fatalf("unexpected profile update = (%q, %q)", profiles.userID, profiles.text)
	}
}

func TestUpdateCustomStatusRejectsMissingProfile(t *testing.T) {
	uc := &UserUsecase{
		customStatusRepo: &customStatusRepositoryFake{err: sql.ErrNoRows},
		presenceRepo:     &presenceRepositoryFake{presence: &domain.Presence{Status: "online"}},
	}

	_, err := uc.UpdateCustomStatus(context.Background(), "missing-user", "studying")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrProfileNotFound)
	}
}

func TestCanViewCustomStatus(t *testing.T) {
	tests := []struct {
		name    string
		viewer  string
		target  string
		roomID  string
		friends bool
		members map[string]bool
		want    string
	}{
		{"owner", "user-1", "user-1", "", false, nil, "studying"},
		{"accepted friend", "user-1", "user-2", "", true, nil, "studying"},
		{"non-friend", "user-1", "user-2", "", false, nil, ""},
		{"both room members", "user-1", "user-2", "room-1", false, map[string]bool{"user-1": true, "user-2": true}, "studying"},
		{"viewer is not room member", "user-1", "user-2", "room-1", false, map[string]bool{"user-1": false, "user-2": true}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presence := &domain.Presence{Status: "online", CustomText: "studying"}
			uc := &UserUsecase{
				friendshipRepo: acceptedFriendsFake{accepted: tt.friends},
				roomClient:     roomClientFake{members: tt.members},
			}

			uc.filterCustomStatus(context.Background(), tt.viewer, tt.target, tt.roomID, presence)

			if presence.CustomText != tt.want {
				t.Fatalf("custom text = %q, want %q", presence.CustomText, tt.want)
			}
			if presence.Status != "online" {
				t.Fatalf("status = %q, want online", presence.Status)
			}
		})
	}
}
