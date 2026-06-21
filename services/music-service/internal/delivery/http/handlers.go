package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/worktogether/services/music-service/internal/domain"
	"github.com/worktogether/services/music-service/internal/usecase"
)

type MusicHandler struct {
	usecase *usecase.MusicUsecase
}

func NewMusicHandler(u *usecase.MusicUsecase) *MusicHandler {
	return &MusicHandler{usecase: u}
}

func (h *MusicHandler) ExtractYoutube(c *gin.Context) {
	var req domain.AddTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Thiếu thông tin URL bài hát.",
			},
		})
		return
	}

	track, err := h.usecase.ExtractTrackMetadata(c.Request.Context(), req.SourceURL)
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

func (h *MusicHandler) UploadAudio(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Không tìm thấy file tải lên.",
			},
		})
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	artist := c.PostForm("artist")

	track, err := h.usecase.UploadAudioFile(c.Request.Context(), file, header.Size, header.Filename, title, artist)
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

func (h *MusicHandler) GetTrack(c *gin.Context) {
	id := c.Param("id")
	track, err := h.usecase.GetTrack(c.Request.Context(), id)
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

	if track == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Không tìm thấy bài hát.",
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

func (h *MusicHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	tracks, err := h.usecase.SearchTracks(c.Request.Context(), keyword)
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

func (h *MusicHandler) LogPlayback(c *gin.Context) {
	var body struct {
		RoomID  string `json:"room_id"`
		TrackID string `json:"track_id"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		// Fallback to form data if JSON binding fails
		body.RoomID = c.PostForm("room_id")
		body.TrackID = c.PostForm("track_id")
	}

	if body.RoomID == "" || body.TrackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Thiếu room_id hoặc track_id.",
			},
		})
		return
	}

	err := h.usecase.LogPlayback(c.Request.Context(), body.RoomID, body.TrackID)
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
		"data":    map[string]string{"message": "Đã ghi nhận lịch sử phát nhạc."},
		"error":   nil,
	})
}

func (h *MusicHandler) GetHistory(c *gin.Context) {
	roomID := c.Param("room_id")
	tracks, err := h.usecase.GetRoomHistory(c.Request.Context(), roomID)
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

func (h *MusicHandler) GetLyrics(c *gin.Context) {
	trackID := c.Param("track_id")
	content, err := h.usecase.GetLyrics(c.Request.Context(), trackID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "LYRICS_NOT_FOUND",
				"message": "Chưa có lời.",
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"track_id": trackID,
			"content":  content,
		},
		"error": nil,
	})
}

func (h *MusicHandler) SaveLyrics(c *gin.Context) {
	trackID := c.Param("track_id")
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}
	if err := h.usecase.SaveLyrics(c.Request.Context(), trackID, req.Content); err != nil {
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
		"data":    gin.H{"message": "Cập nhật thành công."},
		"error":   nil,
	})
}

func (h *MusicHandler) GetBookmarks(c *gin.Context) {
	roomID := c.Param("room_id")
	list, err := h.usecase.GetBookmarks(c.Request.Context(), roomID)
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

func (h *MusicHandler) SaveBookmark(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := c.GetString("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-Id")
	}
	if userID == "" {
		userID = "anonymous"
	}

	var req struct {
		TrackID    string `json:"track_id" binding:"required"`
		PositionMS int    `json:"position_ms" binding:"required"`
		Note       string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	b := &domain.Bookmark{
		ID:         uuid.NewString(),
		RoomID:     roomID,
		UserID:     userID,
		TrackID:    req.TrackID,
		PositionMS: req.PositionMS,
		Note:       req.Note,
	}

	if err := h.usecase.SaveBookmark(c.Request.Context(), b); err != nil {
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
		"data":    b,
		"error":   nil,
	})
}

func (h *MusicHandler) DeleteBookmark(c *gin.Context) {
	id := c.Param("id")
	if err := h.usecase.DeleteBookmark(c.Request.Context(), id); err != nil {
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
		"data":    gin.H{"message": "Đã xóa bookmark thành công."},
		"error":   nil,
	})
}
