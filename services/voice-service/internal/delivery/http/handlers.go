package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	roomv1 "github.com/worktogether/services/voice-service/api/v1"
	"github.com/worktogether/services/voice-service/internal/domain"
	"github.com/worktogether/services/voice-service/internal/usecase"
)

type VoiceHandler struct {
	usecase     *usecase.VoiceUsecase
	roomClient  roomv1.RoomInternalServiceClient
	livekitURL  string
	keyProvider auth.KeyProvider
}

func NewVoiceHandler(uc *usecase.VoiceUsecase, rc roomv1.RoomInternalServiceClient, lkURL, apiKey, apiSecret string) *VoiceHandler {
	return &VoiceHandler{
		usecase:     uc,
		roomClient:  rc,
		livekitURL:  lkURL,
		keyProvider: auth.NewSimpleKeyProvider(apiKey, apiSecret),
	}
}

func (h *VoiceHandler) GetToken(c *gin.Context) {
	roomID := c.Param("room_id")
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(string)
	var req struct {
		ChannelID      string   `json:"channel_id"`
		PublishSources []string `json:"publish_sources"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Yêu cầu token voice không hợp lệ."}})
		return
	}

	// Gọi gRPC room-service xác thực
	res, err := h.roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
		RoomID: roomID,
		UserID: userID,
	})

	if err != nil || res == nil || !res.IsMember || !contains(res.Permissions, "CAN_USE_VOICE") {
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
	if req.ChannelID != "" && req.ChannelID != res.ActiveSubRoomID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Bạn không thuộc phòng thảo luận này."}})
		return
	}

	shortID := userID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	username := "User_" + shortID
	roomName := usecase.LiveKitRoomName(roomID, req.ChannelID)
	token, err := h.usecase.GenerateToken(roomName, userID, username, req.PublishSources)
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
			"expires_in":  int(usecase.TokenTTL.Seconds()),
			"room_name":   roomName,
		},
		"error": nil,
	})
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}

func (h *VoiceHandler) HandleWebhook(c *gin.Context) {
	event, err := webhook.ReceiveWebhookEvent(c.Request, h.keyProvider)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	req := domain.LiveKitWebhookRequest{
		ID:        event.Id,
		Event:     event.Event,
		CreatedAt: event.CreatedAt,
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
			SID:      event.Participant.Sid,
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
