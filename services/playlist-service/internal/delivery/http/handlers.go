package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	roomv1 "github.com/worktogether/services/playlist-service/api/v1"
	"github.com/worktogether/services/playlist-service/internal/domain"
	"github.com/worktogether/services/playlist-service/internal/usecase"
)

type PlaylistHandler struct {
	usecase    *usecase.PlaylistUsecase
	roomClient roomv1.RoomInternalServiceClient
}

func NewPlaylistHandler(u *usecase.PlaylistUsecase, roomClient roomv1.RoomInternalServiceClient) *PlaylistHandler {
	return &PlaylistHandler{usecase: u, roomClient: roomClient}
}

func hasPermission(permissions []string, expected string) bool {
	for _, permission := range permissions {
		if strings.EqualFold(permission, expected) {
			return true
		}
	}
	return false
}
func (h *PlaylistHandler) authorizeRoom(c *gin.Context, roomID, permission string) bool {
	member, err := h.roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{RoomID: roomID, UserID: c.GetString("userID")})
	if err != nil || member == nil || !member.IsMember || (permission != "" && !hasPermission(member.Permissions, permission)) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Bạn không có quyền thao tác playlist của phòng này."}})
		return false
	}
	return true
}
func (h *PlaylistHandler) authorizePlaylist(c *gin.Context, playlistID, permission string) bool {
	p, err := h.usecase.GetPlaylist(c.Request.Context(), playlistID)
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "PLAYLIST_NOT_FOUND"}})
		return false
	}
	if p.RoomID != nil {
		return h.authorizeRoom(c, *p.RoomID, permission)
	}
	if p.UserID == nil || *p.UserID != c.GetString("userID") {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN"}})
		return false
	}
	return true
}
func (h *PlaylistHandler) authorizeTrackVote(c *gin.Context, trackID string) bool {
	p, err := h.usecase.GetPlaylistByTrackID(c.Request.Context(), trackID)
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "TRACK_NOT_FOUND"}})
		return false
	}
	if p.RoomID != nil {
		return h.authorizeRoom(c, *p.RoomID, "CAN_CHAT")
	}
	if p.UserID == nil || *p.UserID != c.GetString("userID") {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN"}})
		return false
	}
	return true
}

func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	var req domain.CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Thông tin tạo playlist không hợp lệ.",
			},
		})
		return
	}

	userID := c.GetString("userID")
	if req.RoomID != nil && !h.authorizeRoom(c, *req.RoomID, "CAN_MANAGE_PLAYLIST") {
		return
	}

	var userPointer *string
	if req.RoomID == nil {
		userPointer = &userID
	}

	p, err := h.usecase.CreatePlaylist(c.Request.Context(), req.Name, req.RoomID, userPointer)
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

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    p,
		"error":   nil,
	})
}

func (h *PlaylistHandler) GetRoomPlaylists(c *gin.Context) {
	roomID := c.Param("room_id")
	if !h.authorizeRoom(c, roomID, "") {
		return
	}
	playlists, err := h.usecase.GetRoomPlaylists(c.Request.Context(), roomID)
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
		"data":    playlists,
		"error":   nil,
	})
}

func (h *PlaylistHandler) GetUserPlaylists(c *gin.Context) {
	userID := c.GetString("userID")
	playlists, err := h.usecase.GetUserPlaylists(c.Request.Context(), userID)
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
		"data":    playlists,
		"error":   nil,
	})
}

func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizePlaylist(c, id, "CAN_MANAGE_PLAYLIST") {
		return
	}
	err := h.usecase.DeletePlaylist(c.Request.Context(), id)
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
		"data":    map[string]string{"message": "Đã xóa danh sách nhạc thành công."},
		"error":   nil,
	})
}

func (h *PlaylistHandler) AddTrack(c *gin.Context) {
	playlistID := c.Param("id")
	if !h.authorizePlaylist(c, playlistID, "CAN_MANAGE_PLAYLIST") {
		return
	}
	var req domain.AddTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Dữ liệu bài hát không hợp lệ.",
			},
		})
		return
	}

	userID := c.GetString("userID")

	track, err := h.usecase.AddTrack(c.Request.Context(), playlistID, &req, userID)
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
		"data":    track,
		"error":   nil,
	})
}

func (h *PlaylistHandler) GetTracks(c *gin.Context) {
	playlistID := c.Param("id")
	if !h.authorizePlaylist(c, playlistID, "") {
		return
	}
	tracks, err := h.usecase.GetPlaylistTracks(c.Request.Context(), playlistID)
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
		"data":    tracks,
		"error":   nil,
	})
}

func (h *PlaylistHandler) RemoveTrack(c *gin.Context) {
	playlistID := c.Param("id")
	if !h.authorizePlaylist(c, playlistID, "CAN_MANAGE_PLAYLIST") {
		return
	}
	itemID := c.Param("item_id")

	err := h.usecase.RemoveTrack(c.Request.Context(), playlistID, itemID)
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
		"data":    map[string]string{"message": "Đã xóa bài hát khỏi danh sách nhạc."},
		"error":   nil,
	})
}

func (h *PlaylistHandler) MoveTrack(c *gin.Context) {
	playlistID := c.Param("id")
	if !h.authorizePlaylist(c, playlistID, "CAN_MANAGE_PLAYLIST") {
		return
	}
	itemID := c.Param("item_id")
	var req domain.MoveTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Vị trí mới không hợp lệ.",
			},
		})
		return
	}

	err := h.usecase.MoveTrack(c.Request.Context(), playlistID, itemID, *req.NewPosition)
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
		"data":    map[string]string{"message": "Sắp xếp lại bài hát thành công."},
		"error":   nil,
	})
}

func (h *PlaylistHandler) VoteTrack(c *gin.Context) {
	itemID := c.Param("item_id")
	if !h.authorizeTrackVote(c, itemID) {
		return
	}
	var req domain.VoteTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Dữ liệu vote không hợp lệ.",
			},
		})
		return
	}

	userID := c.GetString("userID")

	err := h.usecase.VoteTrack(c.Request.Context(), itemID, userID, req.VoteType)
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
		"data":    map[string]string{"message": "Đã ghi nhận bình chọn thành công."},
		"error":   nil,
	})
}
