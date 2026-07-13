package usecase

import (
	"context"
	"encoding/json"
	"fmt"
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

const TokenTTL = 10 * time.Minute

var allowedPublishSources = map[string]struct{}{
	"microphone":   {},
	"camera":       {},
	"screen_share": {},
}

func LiveKitRoomName(roomID, channelID string) string {
	if channelID == "" {
		return "room:" + roomID + ":main"
	}
	return "room:" + roomID + ":sub:" + channelID
}

func (u *VoiceUsecase) GenerateToken(roomName, userID, username string, publishSources []string) (string, error) {
	if len(publishSources) == 0 {
		publishSources = []string{"microphone"}
	}
	for _, source := range publishSources {
		if _, ok := allowedPublishSources[source]; !ok {
			return "", fmt.Errorf("unsupported publish source %q", source)
		}
	}

	at := auth.NewAccessToken(u.livekitKey, u.livekitSecret)

	canPublish := true
	canSubscribe := true
	canPublishData := false
	grant := &auth.VideoGrant{
		RoomJoin:          true,
		Room:              roomName,
		CanPublish:        &canPublish,
		CanSubscribe:      &canSubscribe,
		CanPublishData:    &canPublishData,
		CanPublishSources: publishSources,
	}

	at.AddGrant(grant)
	at.SetIdentity(userID)
	at.SetName(username)
	at.SetValidFor(TokenTTL)

	return at.ToJWT()
}

func (u *VoiceUsecase) ProcessWebhook(ctx context.Context, req *domain.LiveKitWebhookRequest) error {
	eventPayload := VoiceEventPayload(req)

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

func VoiceEventPayload(req *domain.LiveKitWebhookRequest) map[string]interface{} {
	return map[string]interface{}{
		"event_id":        req.ID,
		"event":           req.Event,
		"created_at":      req.CreatedAt,
		"room_id":         req.Room.Name,
		"room_sid":        req.Room.SID,
		"user_id":         req.Participant.Identity,
		"participant_sid": req.Participant.SID,
	}
}
