package usecase

import (
	"strings"
	"testing"
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
