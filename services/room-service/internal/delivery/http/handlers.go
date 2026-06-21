package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/room-service/internal/domain"
	"github.com/worktogether/services/room-service/internal/usecase"
)

type RoomHandler struct {
	usecase *usecase.RoomUsecase
}

func NewRoomHandler(uc *usecase.RoomUsecase) *RoomHandler {
	return &RoomHandler{usecase: uc}
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req domain.CreateRoomRequest
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

	rm, err := h.usecase.CreateRoom(c.Request.Context(), userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "CREATE_ROOM_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    rm,
		"error":   nil,
	})
}

func (h *RoomHandler) GetRoomByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": "Thiếu room_id",
			},
		})
		return
	}

	rm, err := h.usecase.GetRoomByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_ROOM_FAILED",
				"message": err.Error(),
			},
		})
		return
	}
	if rm == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ROOM_NOT_FOUND",
				"message": "Không tìm thấy phòng này.",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rm,
		"error":   nil,
	})
}

func (h *RoomHandler) GetRoomByInviteCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": "Thieu invite code",
			},
		})
		return
	}

	rm, err := h.usecase.GetRoomByInviteCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_ROOM_FAILED",
				"message": err.Error(),
			},
		})
		return
	}
	if rm == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ROOM_NOT_FOUND",
				"message": "Khong tim thay phong voi ma moi nay.",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rm,
		"error":   nil,
	})
}

func (h *RoomHandler) GetRooms(c *gin.Context) {
	search := c.DefaultQuery("search", "")
	list, err := h.usecase.GetRooms(c.Request.Context(), search, 20, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "GET_ROOMS_FAILED",
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

func (h *RoomHandler) GetMembers(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")

	members, err := h.usecase.ListMembers(c.Request.Context(), userID.(string), roomID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "GET_MEMBERS_FAILED"
		if errors.Is(err, usecase.ErrNotMember) {
			status = http.StatusForbidden
			code = "FORBIDDEN"
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

	data := make([]gin.H, 0, len(members))
	for _, member := range members {
		data = append(data, gin.H{
			"id":          member.ID,
			"room_id":     member.RoomID,
			"user_id":     member.UserID,
			"role_id":     member.RoleID,
			"role_type":   member.RoleType,
			"joined_at":   member.JoinedAt,
			"permissions": h.usecase.ResolvePermissions(c.Request.Context(), member),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"error":   nil,
	})
}

func (h *RoomHandler) JoinRoom(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")

	var req domain.JoinRoomRequest
	_ = c.ShouldBindJSON(&req) // Mật khẩu có thể có hoặc không

	member, err := h.usecase.JoinRoom(c.Request.Context(), userID.(string), roomID, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		code := "JOIN_ROOM_FAILED"

		if errors.Is(err, usecase.ErrIncorrectPassword) {
			status = http.StatusForbidden
			code = "INCORRECT_PASSWORD"
		} else if errors.Is(err, usecase.ErrBannedFromRoom) {
			status = http.StatusForbidden
			code = "BANNED"
		} else if errors.Is(err, usecase.ErrAlreadyMember) {
			status = http.StatusBadRequest
			code = "ALREADY_MEMBER"
		} else if errors.Is(err, usecase.ErrRoomNotFound) {
			status = http.StatusNotFound
			code = "ROOM_NOT_FOUND"
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
		"data": gin.H{
			"id":          member.ID,
			"room_id":     member.RoomID,
			"user_id":     member.UserID,
			"role_id":     member.RoleID,
			"role_type":   member.RoleType,
			"joined_at":   member.JoinedAt,
			"role":        member.RoleType,
			"permissions": h.usecase.ResolvePermissions(c.Request.Context(), member),
		},
		"error": nil,
	})
}

func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")

	err := h.usecase.LeaveRoom(c.Request.Context(), userID.(string), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "LEAVE_ROOM_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã rời khỏi phòng.",
		},
		"error": nil,
	})
}

func (h *RoomHandler) CreateRole(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")

	var req domain.CreateRoleRequest
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

	role, err := h.usecase.CreateRole(c.Request.Context(), userID.(string), roomID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "CREATE_ROLE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    role,
		"error":   nil,
	})
}

func (h *RoomHandler) AssignRole(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")
	targetUserID := c.Param("user_id")

	var req domain.AssignRoleRequest
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

	err := h.usecase.AssignRole(c.Request.Context(), userID.(string), roomID, targetUserID, req.RoleID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ASSIGN_ROLE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã gán vai trò thành công.",
		},
		"error": nil,
	})
}

func (h *RoomHandler) KickMember(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")
	targetUserID := c.Param("user_id")

	err := h.usecase.KickMember(c.Request.Context(), userID.(string), roomID, targetUserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "KICK_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã trục xuất thành viên.",
		},
		"error": nil,
	})
}

func (h *RoomHandler) BanMember(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")
	targetUserID := c.Param("user_id")

	var req domain.BanRequest
	_ = c.ShouldBindJSON(&req)

	err := h.usecase.BanMember(c.Request.Context(), userID.(string), roomID, targetUserID, req.Reason)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAN_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Đã cấm thành viên vĩnh viễn.",
		},
		"error": nil,
	})
}

func (h *RoomHandler) UpdateRoomSettings(c *gin.Context) {
	userID, _ := c.Get("userID")
	roomID := c.Param("id")

	var req struct {
		Name           string  `json:"name" binding:"required,min=3,max=100"`
		Description    string  `json:"description" binding:"max=500"`
		AddMusicPolicy string  `json:"add_music_policy" binding:"required,oneof=all nobody dj_only"`
		AvatarURL      *string `json:"avatar_url"`
		Rules          *string `json:"rules"`
		Theme          string  `json:"theme"`
	}

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

	rm, err := h.usecase.UpdateRoomSettings(c.Request.Context(), userID.(string), roomID, req.Name, req.Description, req.AddMusicPolicy, req.AvatarURL, req.Rules, req.Theme)
	if err != nil {
		status := http.StatusInternalServerError
		code := "UPDATE_SETTINGS_FAILED"
		if errors.Is(err, usecase.ErrUnauthorized) {
			status = http.StatusForbidden
			code = "FORBIDDEN"
		} else if errors.Is(err, usecase.ErrRoomNotFound) {
			status = http.StatusNotFound
			code = "ROOM_NOT_FOUND"
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
		"data":    rm,
		"error":   nil,
	})
}

func (h *RoomHandler) CreateSubRoom(c *gin.Context) {
	parentID := c.Param("id")
	userID, _ := c.Get("userID")

	var req struct {
		Name        string `json:"name" binding:"required,min=3,max=100"`
		Description string `json:"description" binding:"max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	sub, err := h.usecase.CreateSubRoom(c.Request.Context(), userID.(string), parentID, req.Name, req.Description)
	if err != nil {
		status := http.StatusInternalServerError
		code := "CREATE_SUBROOM_FAILED"
		if errors.Is(err, usecase.ErrUnauthorized) {
			status = http.StatusForbidden
			code = "FORBIDDEN"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    sub,
	})
}

func (h *RoomHandler) GetSubRooms(c *gin.Context) {
	parentID := c.Param("id")

	subs, err := h.usecase.GetSubRooms(c.Request.Context(), parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "GET_SUBROOMS_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subs,
	})
}

func (h *RoomHandler) MoveMember(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.Param("user_id")
	requesterID, _ := c.Get("userID")

	var req struct {
		SubRoomID *string `json:"sub_room_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PARAMETERS",
				"message": err.Error(),
			},
		})
		return
	}

	err := h.usecase.MoveMember(c.Request.Context(), requesterID.(string), roomID, userID, req.SubRoomID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "MOVE_MEMBER_FAILED"
		if errors.Is(err, usecase.ErrUnauthorized) {
			status = http.StatusForbidden
			code = "FORBIDDEN"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Di chuyển thành viên thành công."},
	})
}

func (h *RoomHandler) DeleteRoom(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("userID")
	if err := h.usecase.DeleteRoom(c.Request.Context(), userID, roomID); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Không có quyền thực hiện.",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ROOM_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    "Xóa phòng thành công.",
		"error":   nil,
	})
}

func (h *RoomHandler) MuteMember(c *gin.Context) {
	roomID := c.Param("id")
	targetUserID := c.Param("user_id")
	requesterID := c.GetString("userID")
	var req domain.MuteRequest
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
	if err := h.usecase.MuteMember(c.Request.Context(), requesterID, roomID, targetUserID, req.DurationSeconds); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Không có quyền thực hiện.",
				},
			})
			return
		}
		if errors.Is(err, usecase.ErrTargetMemberNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "MEMBER_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		if errors.Is(err, usecase.ErrCannotMuteOwner) {
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ROOM_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    "Mute thành viên thành công.",
		"error":   nil,
	})
}

func (h *RoomHandler) UnmuteMember(c *gin.Context) {
	roomID := c.Param("id")
	targetUserID := c.Param("user_id")
	requesterID := c.GetString("userID")
	if err := h.usecase.UnmuteMember(c.Request.Context(), requesterID, roomID, targetUserID); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Không có quyền thực hiện.",
				},
			})
			return
		}
		if errors.Is(err, usecase.ErrTargetMemberNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "MEMBER_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "ROOM_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    "Unmute thành viên thành công.",
		"error":   nil,
	})
}


