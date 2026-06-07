package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	// Trong thực tế, chúng ta nên kiểm tra chữ ký Livekit-Signature tại đây
	var req domain.LiveKitWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.usecase.ProcessWebhook(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook processed successfully",
	})
}
