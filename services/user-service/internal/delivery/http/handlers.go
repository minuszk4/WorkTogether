package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/user-service/internal/domain"
	"github.com/worktogether/services/user-service/internal/usecase"
)

type UserHandler struct {
	usecase *usecase.UserUsecase
}

func NewUserHandler(uc *usecase.UserUsecase) *UserHandler {
	return &UserHandler{usecase: uc}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	// Support cả 2 patterns: /users/:id/profile và /users/profile/:id
	id := c.Param("id")
	if id == "" {
		// Fallback: lấy userID của chính người đang đăng nhập từ JWT middleware
		val, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Không tìm thấy thông tin đăng nhập.",
				},
			})
			return
		}
		id = val.(string)
	}

	p, presence, err := h.usecase.GetProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_PROFILE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":           p.ID,
			"username":     p.ID, // user-service chưa có username, dùng ID tạm
			"display_name": p.DisplayName,
			"avatar_url":   p.AvatarURL,
			"bio":          p.Bio,
			"presence":     presence,
			"created_at":   p.CreatedAt,
		},
		"error": nil,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)

	var req domain.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	p, err := h.usecase.UpdateProfile(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "UPDATE_PROFILE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    p,
		"error":   nil,
	})
}

func (h *UserHandler) UpdatePresence(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)

	var req domain.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	p, err := h.usecase.UpdatePresence(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "UPDATE_PRESENCE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    p,
		"error":   nil,
	})
}

func (h *UserHandler) SendFriendRequest(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)

	var req domain.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	f, err := h.usecase.SendFriendRequest(c.Request.Context(), id, req.FriendID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "FRIEND_REQUEST_ERROR"
		if errors.Is(err, usecase.ErrSelfFriendRequest) {
			status = http.StatusBadRequest
			code = "INVALID_FRIEND_ID"
		}

		c.JSON(status, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    f,
		"error":   nil,
	})
}

func (h *UserHandler) RespondFriendRequest(c *gin.Context) {
	friendshipID := c.Param("id")
	if friendshipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": "Thiếu mã mối quan hệ.",
			},
		})
		return
	}

	var req domain.FriendResponseAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	err := h.usecase.RespondFriendRequest(c.Request.Context(), friendshipID, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "RESPOND_FRIEND_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã phản hồi lời mời kết bạn.",
		},
		"error": nil,
	})
}

func (h *UserHandler) GetFriends(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)

	status := c.DefaultQuery("status", "ACCEPTED")

	list, err := h.usecase.GetFriends(c.Request.Context(), id, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_FRIENDS_ERROR",
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

func (h *UserHandler) BlockUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)

	var req domain.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	err := h.usecase.BlockUser(c.Request.Context(), id, req.FriendID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BLOCK_USER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã chặn người dùng thành công.",
		},
		"error": nil,
	})
}

func (h *UserHandler) CancelFriendRequest(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)
	friendshipID := c.Param("id")
	if friendshipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": "INVALID_PARAMETERS", "message": "Thiếu mã lời mời kết bạn."},
		})
		return
	}

	err := h.usecase.CancelFriendRequest(c.Request.Context(), id, friendshipID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "CANCEL_FRIEND_REQUEST_ERROR"
		if errors.Is(err, usecase.ErrFriendRequestNotFound) {
			status = http.StatusNotFound
			code = "FRIEND_REQUEST_NOT_FOUND"
		}
		c.JSON(status, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": code, "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Đã hủy lời mời kết bạn."},
		"error":   nil,
	})
}

func (h *UserHandler) Unfriend(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)
	friendshipID := c.Param("id")
	if friendshipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": "INVALID_PARAMETERS", "message": "Thiếu mã mối quan hệ."},
		})
		return
	}

	err := h.usecase.Unfriend(c.Request.Context(), id, friendshipID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "UNFRIEND_ERROR"
		if errors.Is(err, usecase.ErrFriendshipNotActive) {
			status = http.StatusNotFound
			code = "FRIENDSHIP_NOT_FOUND"
		}
		c.JSON(status, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": code, "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Đã hủy kết bạn thành công."},
		"error":   nil,
	})
}

func (h *UserHandler) UnblockUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := userID.(string)
	friendshipID := c.Param("id")
	if friendshipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": "INVALID_PARAMETERS", "message": "Thiếu mã mối quan hệ."},
		})
		return
	}

	err := h.usecase.UnblockUser(c.Request.Context(), id, friendshipID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "UNBLOCK_ERROR"
		if errors.Is(err, usecase.ErrBlockNotFound) {
			status = http.StatusNotFound
			code = "BLOCK_NOT_FOUND"
		}
		c.JSON(status, gin.H{
			"success": false, "data": nil,
			"error": gin.H{"code": code, "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Đã bỏ chặn người dùng thành công."},
		"error":   nil,
	})
}
