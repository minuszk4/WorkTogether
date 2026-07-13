package usecase

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/livekit/protocol/auth"
	"github.com/worktogether/services/voice-service/internal/domain"
)

func TestLiveKitRoomName(t *testing.T) {
	tests := []struct {
		name      string
		roomID    string
		channelID string
		want      string
	}{
		{name: "main room", roomID: "room-123", want: "room:room-123:main"},
		{name: "breakout room", roomID: "room-123", channelID: "sub-456", want: "room:room-123:sub:sub-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LiveKitRoomName(tt.roomID, tt.channelID); got != tt.want {
				t.Fatalf("LiveKitRoomName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateTokenUsesShortTTLAndRestrictedSources(t *testing.T) {
	uc := NewVoiceUsecase("wss://livekit.example.com", "test-key", "test-secret", nil)
	token, err := uc.GenerateToken("room:room-123:main", "user-123", "Vanduc", []string{"microphone", "camera"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	verifier, err := auth.ParseAPIToken(token)
	if err != nil {
		t.Fatalf("ParseAPIToken() error = %v", err)
	}
	claims, err := verifier.Verify("test-secret")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.Video == nil || claims.Video.Room != "room:room-123:main" {
		t.Fatalf("unexpected video grant: %#v", claims.Video)
	}
	if claims.Video.CanPublishData == nil || *claims.Video.CanPublishData {
		t.Fatalf("expected data publishing to be disabled")
	}
	if len(claims.Video.CanPublishSources) != 2 {
		t.Fatalf("publish sources = %v, want microphone and camera", claims.Video.CanPublishSources)
	}
	parsed, err := jwt.ParseSigned(token)
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}
	registered := jwt.Claims{}
	if err := parsed.UnsafeClaimsWithoutVerification(&registered); err != nil {
		t.Fatalf("parse token timestamps: %v", err)
	}
	if registered.Expiry == nil || registered.NotBefore == nil {
		t.Fatal("token timestamps are missing")
	}
	if ttl := registered.Expiry.Time().Sub(registered.NotBefore.Time()); ttl > 10*time.Minute+time.Second {
		t.Fatalf("token TTL = %s, want at most 10m", ttl)
	}
}

func TestGenerateTokenRejectsUnknownPublishSource(t *testing.T) {
	uc := NewVoiceUsecase("wss://livekit.example.com", "test-key", "test-secret", nil)
	if _, err := uc.GenerateToken("room:room-123:main", "user-123", "Vanduc", []string{"admin"}); err == nil {
		t.Fatal("GenerateToken() expected error for unknown source")
	}
}

func TestVoiceEventPayloadIncludesIdempotencyFields(t *testing.T) {
	payload := VoiceEventPayload(&domain.LiveKitWebhookRequest{
		ID:        "event-123",
		Event:     "participant_joined",
		CreatedAt: 1712345678,
		Room:      domain.LiveKitRoom{Name: "room:room-123:main", SID: "RM_123"},
		Participant: domain.LiveKitParticipant{
			Identity: "user-123",
			SID:      "PA_123",
		},
	})

	if payload["event_id"] != "event-123" || payload["room_sid"] != "RM_123" || payload["participant_sid"] != "PA_123" {
		t.Fatalf("missing webhook identity fields: %#v", payload)
	}
	if payload["created_at"] != int64(1712345678) {
		t.Fatalf("created_at = %#v", payload["created_at"])
	}
}
