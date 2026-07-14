package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/worktogether/services/user-service/internal/domain"
)

type customStatusRepositoryFake struct {
	userID string
	text   string
	err    error
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
