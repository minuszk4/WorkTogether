package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/room-service/internal/usecase"
)

func (h *RoomHandler) ListSessionTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{
		{"key": "focus", "name": "Focus sprint", "mode": "focus", "duration_seconds": 1500, "break_seconds": 300},
		{"key": "standup", "name": "Daily standup", "mode": "collaborate", "duration_seconds": 900, "break_seconds": 0},
		{"key": "study", "name": "Study group", "mode": "focus", "duration_seconds": 3000, "break_seconds": 600},
		{"key": "chill", "name": "Music chill", "mode": "chill", "duration_seconds": 0, "break_seconds": 0},
	}, "error": nil})
}

func (h *RoomHandler) GetPersonalActionItems(c *gin.Context) {
	userID, _ := c.Get("userID")
	items, err := h.usecase.GetPersonalActionItems(c.Request.Context(), userID.(string))
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items, "error": nil})
}

func (h *RoomHandler) StartSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Title       string `json:"title" binding:"required,min=3,max=160"`
		Goal        string `json:"goal" binding:"max=1000"`
		TemplateKey string `json:"template_key" binding:"max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSessionBadRequest(c, err)
		return
	}
	session, err := h.usecase.StartSession(c.Request.Context(), userID.(string), c.Param("id"), &usecase.StartSessionInput{Title: req.Title, Goal: req.Goal, TemplateKey: req.TemplateKey})
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": session, "error": nil})
}

func (h *RoomHandler) GetActiveSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	session, err := h.usecase.GetActiveSession(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": session, "error": nil})
}

func (h *RoomHandler) CompleteSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.usecase.CompleteSession(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id")); err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"session_id": c.Param("session_id"), "status": "COMPLETED"}, "error": nil})
}

func (h *RoomHandler) GetSessionWorkspace(c *gin.Context) {
	userID, _ := c.Get("userID")
	workspace, err := h.usecase.GetSessionWorkspace(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id"))
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": workspace, "error": nil})
}

func (h *RoomHandler) AddAgendaItem(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Content  string `json:"content" binding:"required,max=500"`
		Position int    `json:"position" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSessionBadRequest(c, err)
		return
	}
	item, err := h.usecase.AddAgendaItem(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id"), &usecase.AddAgendaInput{Content: req.Content, Position: req.Position})
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item, "error": nil})
}

func (h *RoomHandler) UpdateAgendaItem(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		IsDone bool `json:"is_done"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSessionBadRequest(c, err)
		return
	}
	if err := h.usecase.SetAgendaItemDone(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id"), c.Param("item_id"), req.IsDone); err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": c.Param("item_id"), "is_done": req.IsDone}, "error": nil})
}

func (h *RoomHandler) AddActionItem(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Content    string     `json:"content" binding:"required,max=500"`
		AssigneeID *string    `json:"assignee_id"`
		DueAt      *time.Time `json:"due_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSessionBadRequest(c, err)
		return
	}
	item, err := h.usecase.AddActionItem(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id"), &usecase.AddActionInput{Content: req.Content, AssigneeID: req.AssigneeID, DueAt: req.DueAt})
	if err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item, "error": nil})
}

func (h *RoomHandler) UpdateActionItem(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Status string `json:"status" binding:"required,oneof=OPEN DONE"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeSessionBadRequest(c, err)
		return
	}
	if err := h.usecase.SetActionStatus(c.Request.Context(), userID.(string), c.Param("id"), c.Param("session_id"), c.Param("action_id"), req.Status); err != nil {
		h.writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": c.Param("action_id"), "status": req.Status}, "error": nil})
}

func (h *RoomHandler) writeSessionError(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, "SESSION_FAILED"
	switch {
	case errors.Is(err, usecase.ErrUnauthorized), errors.Is(err, usecase.ErrNotMember):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, usecase.ErrSessionNotFound):
		status, code = http.StatusNotFound, "SESSION_NOT_FOUND"
	case errors.Is(err, usecase.ErrActiveSessionExists):
		status, code = http.StatusConflict, "ACTIVE_SESSION_EXISTS"
	}
	c.JSON(status, gin.H{"success": false, "data": nil, "error": gin.H{"code": code, "message": err.Error()}})
}

func (h *RoomHandler) writeSessionBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"success": false, "data": nil, "error": gin.H{"code": "INVALID_PARAMETERS", "message": err.Error()}})
}
