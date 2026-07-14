package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/worktogether/services/user-service/internal/domain"
)

type customStatusRepositoryFake struct {
	userID string
	text   string
}

func (r *customStatusRepositoryFake) UpdateCustomStatus(_ context.Context, userID, text string) error {
	r.userID = userID
	r.text = text
	return nil
}

type presenceRepositoryFake struct {
	presence *domain.Presence
}

func (r *presenceRepositoryFake) GetPresence(context.Context, string) (*domain.Presence, error) {
	return r.presence, nil
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
