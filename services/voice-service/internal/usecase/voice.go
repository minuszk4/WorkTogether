package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/voice-service/internal/domain"
)

type VoiceUsecase struct {
	livekitURL    string
	livekitKey    string
	livekitSecret string
	rdb           *redis.Client
}

func NewVoiceUsecase(url, key, secret string, rdb *redis.Client) *VoiceUsecase {
	return &VoiceUsecase{
		livekitURL:    url,
		livekitKey:    key,
		livekitSecret: secret,
		rdb:           rdb,
	}
}

func (u *VoiceUsecase) GenerateToken(ctx context.Context, roomID, userID, username string) (string, error) {
	at := auth.NewAccessToken(u.livekitKey, u.livekitSecret)
	
	// Cấp quyền kết nối RTC Room
	canPublish := true
	canSubscribe := true
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomID,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}
	
	at.AddGrant(grant)
	at.SetIdentity(userID)
	at.SetName(username)
	at.SetValidFor(2 * time.Hour) // Token có thời hạn 2 tiếng

	return at.ToJWT()
}

func (u *VoiceUsecase) ProcessWebhook(ctx context.Context, req *domain.LiveKitWebhookRequest) error {
	// Publish sự kiện thoại vào Redis Streams
	eventPayload := map[string]interface{}{
		"event":   req.Event,
		"room_id": req.Room.Name,
		"user_id": req.Participant.Identity,
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return err
	}

	streamKey := "stream:voice_events"
	return u.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"payload": string(payloadBytes),
		},
	}).Err()
}
