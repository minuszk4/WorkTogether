package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/room-service/internal/domain"
	"github.com/worktogether/services/room-service/internal/usecase"
)

func (h *RoomHandler) AdminListRooms(c *gin.Context) {
	rooms, err := h.usecase.AdminListRooms(c.Request.Context(), isAdmin(c))
	if err != nil {
		h.writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rooms, "error": nil})
}

func (h *RoomHandler) AdminUpdateRoom(c *gin.Context) {
	var req struct {
		Name           string  `json:"name" binding:"required,min=3,max=100"`
		Description    string  `json:"description" binding:"max=500"`
		Privacy        string  `json:"privacy" binding:"required,oneof=public private friends"`
		AddMusicPolicy string  `json:"add_music_policy" binding:"required,oneof=all nobody dj_only"`
		AvatarURL      *string `json:"avatar_url"`
		Rules          *string `json:"rules"`
		Theme          string  `json:"theme"`
		Mode           string  `json:"mode" binding:"omitempty,oneof=chill focus collaborate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "data": nil, "error": gin.H{"code": "INVALID_PARAMETERS", "message": err.Error()}})
		return
	}
	actorID, _ := c.Get("userID")
	rm, err := h.usecase.AdminUpdateRoom(c.Request.Context(), isAdmin(c), actorID.(string), c.Param("id"), &domain.Room{Name: req.Name, Description: req.Description, Privacy: req.Privacy, AddMusicPolicy: req.AddMusicPolicy, AvatarURL: req.AvatarURL, Rules: req.Rules, Theme: req.Theme, Mode: req.Mode})
	if err != nil {
		h.writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rm, "error": nil})
}

func (h *RoomHandler) AdminDeleteRoom(c *gin.Context) {
	actorID, _ := c.Get("userID")
	if err := h.usecase.AdminDeleteRoom(c.Request.Context(), isAdmin(c), actorID.(string), c.Param("id")); err != nil {
		h.writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": c.Param("id")}, "error": nil})
}

func isAdmin(c *gin.Context) bool {
	value, _ := c.Get("isAdmin")
	enabled, _ := value.(bool)
	return enabled
}
func (h *RoomHandler) writeAdminError(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, "ADMIN_FAILED"
	if errors.Is(err, usecase.ErrUnauthorized) {
		status, code = http.StatusForbidden, "FORBIDDEN"
	}
	if errors.Is(err, usecase.ErrRoomNotFound) {
		status, code = http.StatusNotFound, "ROOM_NOT_FOUND"
	}
	c.JSON(status, gin.H{"success": false, "data": nil, "error": gin.H{"code": code, "message": err.Error()}})
}
