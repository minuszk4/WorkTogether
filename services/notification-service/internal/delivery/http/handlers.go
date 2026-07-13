package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/worktogether/pkg/env"
	"github.com/worktogether/services/notification-service/internal/domain"
	"github.com/worktogether/services/notification-service/internal/usecase"
)

type NotificationHandler struct {
	usecase   *usecase.NotificationUsecase
	jwtSecret string
}

func NewNotificationHandler(u *usecase.NotificationUsecase, jwtSecret string) *NotificationHandler {
	return &NotificationHandler{
		usecase:   u,
		jwtSecret: jwtSecret,
	}
}

func (h *NotificationHandler) requireAdmin(c *gin.Context) bool {
	if isAdmin, _ := c.Get("isAdmin"); isAdmin == true {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN"}})
	return false
}

func sseTokenSubject(tokenString, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "access_token" {
		return "", fmt.Errorf("invalid token type")
	}
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("missing token subject")
	}
	return userID, nil
}

func (h *NotificationHandler) ServeSSE(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		allowedOrigins := env.GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4200")
		allowed := false
		for _, o := range strings.Split(allowedOrigins, ",") {
			if origin == o {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "CORS Origin không hợp lệ"})
			return
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực"})
		return
	}

	userID, err := sseTokenSubject(tokenStr, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc hết hạn"})
		return
	}

	// Cấu hình Headers cho SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	// Đăng ký kênh truyền dữ liệu
	notiChan := h.usecase.RegisterStream(userID)
	defer h.usecase.UnregisterStream(userID, notiChan)

	c.SSEvent("message", "Connected to WorkTogether Notifications")
	c.Writer.Flush()

	// Ticker để gửi Heartbeat giữ connection online
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Lắng nghe đóng kết nối từ client
	clientDone := c.Request.Context().Done()

	for {
		select {
		case <-clientDone:
			return
		case <-ticker.C:
			// Gửi ping comment để duy trì kết nối
			c.SSEvent("ping", "keep-alive")
			c.Writer.Flush()
		case noti, ok := <-notiChan:
			if !ok {
				return
			}
			notiBytes, err := json.Marshal(noti)
			if err == nil {
				c.SSEvent("notification", string(notiBytes))
				c.Writer.Flush()
			}
		}
	}
}

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	list, err := h.usecase.GetNotifications(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
		"error":   nil,
	})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	err := h.usecase.MarkRead(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    map[string]string{"message": "Đã đánh dấu thông báo là đã đọc."},
		"error":   nil,
	})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetString("userID")

	err := h.usecase.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    map[string]string{"message": "Đã đánh dấu tất cả thông báo là đã đọc."},
		"error":   nil,
	})
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("userID")

	count, err := h.usecase.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    map[string]int{"unread_count": count},
		"error":   nil,
	})
}

// REST trigger để kích hoạt test thông báo
func (h *NotificationHandler) TriggerNotification(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	var req domain.TriggerNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	n := &domain.Notification{
		ReceiverID: req.ReceiverID,
		SenderID:   c.GetString("userID"),
		Type:       req.Type,
		Content:    req.Content,
		IsRead:     false,
		CreatedAt:  time.Now(),
	}

	err := h.usecase.SendNotification(c.Request.Context(), n)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã kích hoạt thông báo"})
}
