package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"github.com/worktogether/pkg/env"
	roomv1 "github.com/worktogether/services/voice-service/api/v1"
	"github.com/worktogether/services/voice-service/internal/domain"
	"github.com/worktogether/services/voice-service/internal/usecase"
)

type VoiceHandler struct {
	usecase    *usecase.VoiceUsecase
	roomClient roomv1.RoomInternalServiceClient
	livekitURL string
}

func NewVoiceHandler(uc *usecase.VoiceUsecase, rc roomv1.RoomInternalServiceClient, lkURL string) *VoiceHandler {
	return &VoiceHandler{
		usecase:    uc,
		roomClient: rc,
		livekitURL: lkURL,
	}
}

func (h *VoiceHandler) GetToken(c *gin.Context) {
	roomID := c.Param("room_id")
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(string)

	// Gọi gRPC room-service xác thực
	res, err := h.roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
		RoomID: roomID,
		UserID: userID,
	})

	if err != nil || res == nil || !res.IsMember {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Bạn không có quyền tham gia kênh thoại của phòng này.",
			},
		})
		return
	}

	// Độc lập sinh token
	username := "User_" + userID[:8]
	token, err := h.usecase.GenerateToken(c.Request.Context(), roomID, userID, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GENERATE_TOKEN_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"livekit_url": h.livekitURL,
			"token":       token,
		},
		"error": nil,
	})
}

func (h *VoiceHandler) HandleWebhook(c *gin.Context) {
	apiKey := env.GetEnv("LIVEKIT_API_KEY", "devkey")
	apiSecret := env.GetEnv("LIVEKIT_API_SECRET", "your_super_secret_livekit_key")
	provider := auth.NewSimpleKeyProvider(apiKey, apiSecret)

	event, err := webhook.ReceiveWebhookEvent(c.Request, provider)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	req := domain.LiveKitWebhookRequest{
		Event: event.Event,
	}
	if event.Room != nil {
		req.Room = domain.LiveKitRoom{
			Name: event.Room.Name,
			SID:  event.Room.Sid,
		}
	}
	if event.Participant != nil {
		req.Participant = domain.LiveKitParticipant{
			Identity: event.Participant.Identity,
			State:    event.Participant.State.String(),
			JoinedAt: event.Participant.JoinedAt,
		}
	}

	err = h.usecase.ProcessWebhook(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook processed successfully",
	})
}
