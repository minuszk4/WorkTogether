package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	roomv1 "github.com/worktogether/services/playback-service/api/v1"
	"github.com/worktogether/services/playback-service/internal/domain"
	"github.com/worktogether/services/playback-service/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Cấu hình CORS ở Nginx Gateway
	},
}

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	RoomID string
	UserID string
	Token  string
}

type Hub struct {
	usecase    *usecase.PlaybackUsecase
	jwtSecret  string
	rooms      map[string]map[*Client]bool // map room_id -> list client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *domain.WSMessage
	roomClient roomv1.RoomInternalServiceClient
	mutex      sync.RWMutex
}

func NewHub(u *usecase.PlaybackUsecase, rc roomv1.RoomInternalServiceClient, jwtSecret string) *Hub {
	return &Hub{
		usecase:    u,
		jwtSecret:  jwtSecret,
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *domain.WSMessage),
		roomClient: rc,
	}
}

func (h *Hub) roomMembership(ctx context.Context, roomID, userID string) *roomv1.VerifyRoomMemberResponse {
	if h.roomClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	member, err := h.roomClient.VerifyRoomMember(ctx, &roomv1.VerifyRoomMemberRequest{RoomID: roomID, UserID: userID})
	if err != nil || member == nil || !member.IsMember {
		return nil
	}
	return member
}

func (h *Hub) canControlPlayback(ctx context.Context, roomID, userID string) bool {
	member := h.roomMembership(ctx, roomID, userID)
	if member == nil {
		return false
	}
	for _, permission := range member.Permissions {
		if strings.EqualFold(permission, "CAN_CONTROL_PLAYBACK") {
			return true
		}
	}
	return false
}

func (h *Hub) Run() {
	// Khởi chạy vòng lặp kiểm tra trạng thái phát nhạc của tất cả các phòng có người kết nối
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			ctx := context.Background()

			h.mutex.RLock()
			roomIDs := make([]string, 0, len(h.rooms))
			for rID := range h.rooms {
				roomIDs = append(roomIDs, rID)
			}
			h.mutex.RUnlock()

			for _, roomID := range roomIDs {
				h.checkRoomPlayback(ctx, roomID)
			}
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			h.mutex.Unlock()
			log.Printf("Client %s kết nối vào phòng playback %s\n", client.UserID, client.RoomID)

			// Gửi ngay trạng thái playback hiện tại cho client mới kết nối
			ctx := context.Background()
			state, err := h.usecase.GetOrCreateState(ctx, client.RoomID)
			if err == nil {
				payloadBytes, _ := json.Marshal(state)
				msg := domain.WSMessage{
					Event:   "playback:sync",
					RoomID:  client.RoomID,
					Payload: json.RawMessage(payloadBytes),
				}
				msgBytes, _ := json.Marshal(msg)
				client.Send <- msgBytes
			}

		case client := <-h.unregister:
			h.mutex.Lock()
			if rooms, exists := h.rooms[client.RoomID]; exists {
				if _, ok := rooms[client]; ok {
					delete(rooms, client)
					close(client.Send)
					log.Printf("Client %s rời phòng playback %s\n", client.UserID, client.RoomID)
				}
				if len(rooms) == 0 {
					delete(h.rooms, client.RoomID)
				}
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			clients := h.rooms[message.RoomID]
			if clients != nil {
				msgBytes, err := json.Marshal(message)
				if err == nil {
					for client := range clients {
						select {
						case client.Send <- msgBytes:
						default:
							h.mutex.RUnlock()
							h.mutex.Lock()
							delete(clients, client)
							close(client.Send)
							h.mutex.Unlock()
							h.mutex.RLock()
						}
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Nhận dữ liệu thô và parse Event
		var rawMsg struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(messageBytes, &rawMsg); err != nil {
			continue
		}

		ctx := context.Background()

		// 1. Xử lý đồng bộ NTP-like Time
		if rawMsg.Event == "sync:ping" {
			t2 := time.Now().UnixNano() / int64(time.Millisecond) // Server nhận ping

			var ping domain.PingPayload
			if err := json.Unmarshal(rawMsg.Payload, &ping); err != nil {
				continue
			}

			t3 := time.Now().UnixNano() / int64(time.Millisecond) // Server gửi pong

			pongPayload := domain.PongPayload{
				T1: ping.T1,
				T2: t2,
				T3: t3,
			}
			payloadBytes, _ := json.Marshal(pongPayload)

			resp := domain.WSMessage{
				Event:   "sync:pong",
				RoomID:  c.RoomID,
				Payload: json.RawMessage(payloadBytes),
			}
			respBytes, _ := json.Marshal(resp)
			c.Send <- respBytes
			continue
		}

		if rawMsg.Event == "poll:vote" {
			var voteReq struct {
				TrackID string `json:"track_id"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &voteReq); err != nil {
				continue
			}

			// Ghi nhận vote
			if err := c.Hub.usecase.VoteForTrack(ctx, c.RoomID, voteReq.TrackID); err != nil {
				log.Printf("Lỗi ghi nhận vote: %v\n", err)
				continue
			}

			// Tính toán lại tổng số vote hiện tại và broadcast poll:update
			votes, err := c.Hub.usecase.GetPollVotes(ctx, c.RoomID)
			if err == nil {
				c.Hub.BroadcastToRoom(c.RoomID, "poll:update", votes)
			}
			continue
		}

		// 2. Xử lý lệnh điều khiển Playback
		if rawMsg.Event == "playback:control" {
			var ctrl domain.ControlPayload
			if err := json.Unmarshal(rawMsg.Payload, &ctrl); err != nil {
				continue
			}

			// Guest DJ có thể điều khiển trong thời hạn được cấp; các host/moderator
			// vẫn có thể override bằng quyền CAN_CONTROL_PLAYBACK.
			guestDJ, err := c.Hub.usecase.GetGuestDJ(ctx, c.RoomID)
			if (err != nil || c.UserID != guestDJ) && !c.Hub.canControlPlayback(ctx, c.RoomID, c.UserID) {
				payloadBytes, _ := json.Marshal(map[string]string{"code": "FORBIDDEN", "message": "Bạn không có quyền điều khiển phát nhạc của phòng này."})
				respBytes, _ := json.Marshal(domain.WSMessage{Event: "playback:error", RoomID: c.RoomID, Payload: json.RawMessage(payloadBytes)})
				c.Send <- respBytes
				continue
			}

			// Lưu trạng thái mới vào Redis
			state, err := c.Hub.usecase.UpdateState(ctx, c.RoomID, &ctrl)
			if err != nil {
				log.Printf("Lỗi lưu playback state: %v\n", err)
				continue
			}

			if ctrl.Action == "play" && ctrl.TrackID != "" {
				c.Hub.logHistoryToMusicService(c.RoomID, ctrl.TrackID)
			}

			// Broadcast trạng thái mới tới cả phòng
			payloadBytes, _ := json.Marshal(state)
			c.Hub.broadcast <- &domain.WSMessage{
				Event:   "playback:sync",
				RoomID:  c.RoomID,
				Payload: json.RawMessage(payloadBytes),
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Thêm các message hàng đợi
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServePlaybackWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		tokenStr := c.Query("token")

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực"})
			return
		}

		// Validate token
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(hub.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc hết hạn"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["type"] != "access_token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Loại token không hợp lệ"})
			return
		}

		userID, _ := claims["sub"].(string)
		if hub.roomMembership(c.Request.Context(), roomID, userID) == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của phòng này"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Lỗi nâng cấp WebSocket: %v\n", err)
			return
		}

		client := &Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			RoomID: roomID,
			UserID: userID,
			Token:  tokenStr,
		}

		client.Hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}

func (h *Hub) BroadcastToRoom(roomID string, event string, payload interface{}) {
	payloadBytes, _ := json.Marshal(payload)
	h.broadcast <- &domain.WSMessage{
		Event:   event,
		RoomID:  roomID,
		Payload: json.RawMessage(payloadBytes),
	}
}

type playlistInfo struct {
	ID     string `json:"id"`
	RoomID string `json:"room_id"`
	Name   string `json:"name"`
}

type playlistTrackInfo struct {
	ID           string `json:"id"`
	PlaylistID   string `json:"playlist_id"`
	TrackID      string `json:"track_id"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationMS   int    `json:"duration_ms"`
	SourceURL    string `json:"source_url"`
	Position     int    `json:"position"`
	AddedBy      string `json:"added_by"`
	Votes        int    `json:"votes"`
}

func (h *Hub) getClientToken(roomID string) string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	clients := h.rooms[roomID]
	if clients != nil {
		for client := range clients {
			if client.Token != "" {
				return client.Token
			}
		}
	}
	return ""
}

func (h *Hub) checkRoomPlayback(ctx context.Context, roomID string) {
	state, err := h.usecase.GetOrCreateState(ctx, roomID)
	if err != nil || state == nil || state.State != "playing" || state.DurationMS <= 0 {
		return
	}

	nowMS := time.Now().UnixNano() / int64(time.Millisecond)
	elapsed := nowMS - state.UpdatedAt
	currentPos := int(state.PositionMS) + int(elapsed)

	token := h.getClientToken(roomID)
	if token == "" {
		return
	}

	// 1. Kiểm tra bài hát kết thúc
	if currentPos >= state.DurationMS {
		h.advanceToNextTrack(ctx, roomID, token)
		return
	}

	// 2. Kiểm tra kích hoạt poll ở 30s cuối
	remaining := state.DurationMS - currentPos
	if remaining <= 30000 && remaining > 0 {
		active, err := h.usecase.IsPollActive(ctx, roomID)
		if err == nil && !active {
			h.startPlaylistPoll(ctx, roomID, token)
		}
	}
}

func (h *Hub) startPlaylistPoll(ctx context.Context, roomID string, token string) {
	_ = h.usecase.SetPollActive(ctx, roomID, true)

	playlistID, err := h.fetchDefaultPlaylistID(ctx, roomID, token)
	if err != nil {
		log.Printf("Lỗi lấy default playlist cho phòng %s: %v\n", roomID, err)
		return
	}

	tracks, err := h.fetchPlaylistTracks(ctx, playlistID, token)
	if err != nil || len(tracks) == 0 {
		return
	}

	candidates := make([]interface{}, 0, 3)
	limit := 3
	if len(tracks) < limit {
		limit = len(tracks)
	}

	// Tránh đề xuất chính bài đang phát hiện tại
	state, _ := h.usecase.GetOrCreateState(ctx, roomID)
	currentIndex := -1
	if state != nil {
		for idx, t := range tracks {
			if t.TrackID == state.CurrentTrackID {
				currentIndex = idx
				break
			}
		}
	}

	addedCount := 0
	for i := 0; i < len(tracks) && addedCount < 3; i++ {
		if i == currentIndex {
			continue // Không đưa bài đang phát vào danh sách vote
		}
		candidates = append(candidates, map[string]interface{}{
			"id":            tracks[i].ID, // playlist_track_id
			"track_id":      tracks[i].TrackID,
			"title":         tracks[i].Title,
			"artist":        tracks[i].Artist,
			"thumbnail_url": tracks[i].ThumbnailURL,
			"duration_ms":   tracks[i].DurationMS,
			"source_url":    tracks[i].SourceURL,
		})
		addedCount++
	}

	if len(candidates) > 0 {
		h.BroadcastToRoom(roomID, "poll:start", map[string]interface{}{
			"candidates": candidates,
			"duration":   30000,
		})
	}
}

func (h *Hub) advanceToNextTrack(ctx context.Context, roomID string, token string) {
	votes, _ := h.usecase.GetPollVotes(ctx, roomID)
	playlistID, err := h.fetchDefaultPlaylistID(ctx, roomID, token)
	if err != nil {
		h.usecase.ClearPoll(ctx, roomID)
		return
	}

	tracks, err := h.fetchPlaylistTracks(ctx, playlistID, token)
	if err != nil || len(tracks) == 0 {
		h.stopPlayback(ctx, roomID)
		h.usecase.ClearPoll(ctx, roomID)
		h.BroadcastToRoom(roomID, "poll:end", map[string]interface{}{})
		return
	}

	// Xác định track chiến thắng
	winningTrackID := ""
	maxVotes := -1
	for trackID, voteCount := range votes {
		if voteCount > maxVotes {
			maxVotes = voteCount
			winningTrackID = trackID
		}
	}

	var winningTrack *playlistTrackInfo
	if winningTrackID != "" {
		for _, t := range tracks {
			if t.ID == winningTrackID {
				winningTrack = t
				break
			}
		}
	}

	currentTrackIDInPlaylist := ""
	state, _ := h.usecase.GetOrCreateState(ctx, roomID)
	if state != nil && state.CurrentTrackID != "" {
		for _, t := range tracks {
			if t.TrackID == state.CurrentTrackID {
				currentTrackIDInPlaylist = t.ID
				break
			}
		}
	}

	if winningTrack == nil {
		var candidates []*playlistTrackInfo
		for _, t := range tracks {
			if t.ID != currentTrackIDInPlaylist {
				candidates = append(candidates, t)
			}
		}

		if len(candidates) > 0 {
			userTracksMap := make(map[string][]*playlistTrackInfo)
			var activeUsers []string

			for _, t := range candidates {
				user := t.AddedBy
				if len(userTracksMap[user]) == 0 {
					activeUsers = append(activeUsers, user)
				}
				userTracksMap[user] = append(userTracksMap[user], t)
			}

			for user := range userTracksMap {
				sort.Slice(userTracksMap[user], func(i, j int) bool {
					if userTracksMap[user][i].Votes != userTracksMap[user][j].Votes {
						return userTracksMap[user][i].Votes > userTracksMap[user][j].Votes
					}
					return userTracksMap[user][i].Position < userTracksMap[user][j].Position
				})
			}

			sort.Slice(activeUsers, func(i, j int) bool {
				uI := activeUsers[i]
				uJ := activeUsers[j]

				minPosI := 9999999
				for _, t := range userTracksMap[uI] {
					if t.Position < minPosI {
						minPosI = t.Position
					}
				}

				minPosJ := 9999999
				for _, t := range userTracksMap[uJ] {
					if t.Position < minPosJ {
						minPosJ = t.Position
					}
				}

				return minPosI < minPosJ
			})

			currentUser := ""
			if state != nil && state.CurrentTrackID != "" {
				for _, t := range tracks {
					if t.TrackID == state.CurrentTrackID {
						currentUser = t.AddedBy
						break
					}
				}
			}

			nextUserIndex := 0
			if currentUser != "" {
				currentUserIndex := -1
				for idx, user := range activeUsers {
					if user == currentUser {
						currentUserIndex = idx
						break
					}
				}
				if currentUserIndex != -1 {
					nextUserIndex = (currentUserIndex + 1) % len(activeUsers)
				}
			}

			selectedUser := activeUsers[nextUserIndex]
			winningTrack = userTracksMap[selectedUser][0]
		}
	}

	if winningTrack != nil && winningTrack.ID != currentTrackIDInPlaylist {
		_ = h.moveTrackInPlaylist(ctx, playlistID, winningTrack.ID, 0, token)
	}

	if currentTrackIDInPlaylist != "" {
		_ = h.removeTrackFromPlaylist(ctx, playlistID, currentTrackIDInPlaylist, token)
	}

	newTracks, err := h.fetchPlaylistTracks(ctx, playlistID, token)
	if err != nil || len(newTracks) == 0 {
		h.stopPlayback(ctx, roomID)
		h.usecase.ClearPoll(ctx, roomID)
		h.BroadcastToRoom(roomID, "poll:end", map[string]interface{}{})
		return
	}

	nextTrack := newTracks[0]

	h.logHistoryToMusicService(roomID, nextTrack.TrackID)

	nowMS := time.Now().UnixNano() / int64(time.Millisecond)
	newState := &domain.PlaybackState{
		State:          "playing",
		CurrentTrackID: nextTrack.TrackID,
		PositionMS:     0,
		UpdatedAt:      nowMS,
		Title:          nextTrack.Title,
		Artist:         nextTrack.Artist,
		ThumbnailURL:   nextTrack.ThumbnailURL,
		DurationMS:     nextTrack.DurationMS,
		SourceURL:      nextTrack.SourceURL,
	}

	_, _ = h.usecase.UpdateState(ctx, roomID, &domain.ControlPayload{
		Action:       "play",
		TrackID:      nextTrack.TrackID,
		PositionMS:   0,
		Title:        nextTrack.Title,
		Artist:       nextTrack.Artist,
		ThumbnailURL: nextTrack.ThumbnailURL,
		DurationMS:   nextTrack.DurationMS,
		SourceURL:    nextTrack.SourceURL,
	})

	h.usecase.ClearPoll(ctx, roomID)

	syncBytes, _ := json.Marshal(newState)
	h.BroadcastToRoom(roomID, "playback:sync", json.RawMessage(syncBytes))

	h.BroadcastToRoom(roomID, "poll:end", map[string]interface{}{
		"winner": map[string]interface{}{
			"track_id": nextTrack.TrackID,
			"title":    nextTrack.Title,
		},
	})
}

func (h *Hub) stopPlayback(ctx context.Context, roomID string) {
	nowMS := time.Now().UnixNano() / int64(time.Millisecond)
	newState := &domain.PlaybackState{
		State:          "stopped",
		CurrentTrackID: "",
		PositionMS:     0,
		UpdatedAt:      nowMS,
	}
	_, _ = h.usecase.UpdateState(ctx, roomID, &domain.ControlPayload{
		Action:     "stop",
		TrackID:    "",
		PositionMS: 0,
	})
	syncBytes, _ := json.Marshal(newState)
	h.BroadcastToRoom(roomID, "playback:sync", json.RawMessage(syncBytes))
}

func (h *Hub) fetchDefaultPlaylistID(ctx context.Context, roomID string, token string) (string, error) {
	url := fmt.Sprintf("http://playlist-service:8086/api/v1/playlists/room/%s", roomID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("playlist-service returned status code %d", resp.StatusCode)
	}

	var res struct {
		Success bool            `json:"success"`
		Data    []*playlistInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Data) == 0 {
		return "", fmt.Errorf("no playlist found for room %s", roomID)
	}

	return res.Data[0].ID, nil
}

func (h *Hub) fetchPlaylistTracks(ctx context.Context, playlistID string, token string) ([]*playlistTrackInfo, error) {
	url := fmt.Sprintf("http://playlist-service:8086/api/v1/playlists/%s/tracks", playlistID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("playlist-service returned status code %d", resp.StatusCode)
	}

	var res struct {
		Success bool                 `json:"success"`
		Data    []*playlistTrackInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

func (h *Hub) moveTrackInPlaylist(ctx context.Context, playlistID string, itemID string, newPosition int, token string) error {
	url := fmt.Sprintf("http://playlist-service:8086/api/v1/playlists/%s/tracks/%s/move", playlistID, itemID)

	bodyMap := map[string]interface{}{
		"new_position": newPosition,
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("playlist-service returned status code %d", resp.StatusCode)
	}

	return nil
}

func (h *Hub) removeTrackFromPlaylist(ctx context.Context, playlistID string, itemID string, token string) error {
	url := fmt.Sprintf("http://playlist-service:8086/api/v1/playlists/%s/tracks/%s", playlistID, itemID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("playlist-service returned status code %d", resp.StatusCode)
	}

	return nil
}

func (h *Hub) TriggerTimerPlaybackAction(ctx context.Context, roomID string, action string) error {
	state, err := h.usecase.GetOrCreateState(ctx, roomID)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("room state not initialized")
	}

	ctrl := &domain.ControlPayload{
		Action:       action,
		TrackID:      state.CurrentTrackID,
		PositionMS:   state.PositionMS,
		Title:        state.Title,
		Artist:       state.Artist,
		ThumbnailURL: state.ThumbnailURL,
		DurationMS:   state.DurationMS,
		SourceURL:    state.SourceURL,
	}

	if action == "pause" {
		nowMS := time.Now().UnixNano() / int64(time.Millisecond)
		elapsed := nowMS - state.UpdatedAt
		ctrl.PositionMS = state.PositionMS + int(elapsed)
		if ctrl.PositionMS > state.DurationMS {
			ctrl.PositionMS = state.DurationMS
		}
	} else if action == "resume" {
		ctrl.Action = "play"
	}

	newState, err := h.usecase.UpdateState(ctx, roomID, ctrl)
	if err != nil {
		return err
	}

	h.BroadcastToRoom(roomID, "playback:sync", newState)
	return nil
}

func (h *Hub) logHistoryToMusicService(roomID string, trackID string) {
	go func() {
		payload := map[string]string{
			"room_id":  roomID,
			"track_id": trackID,
		}
		jsonBytes, _ := json.Marshal(payload)
		resp, err := http.Post("http://music-service:8085/api/v1/music/history", "application/json", bytes.NewBuffer(jsonBytes))
		if err != nil {
			log.Printf("[HISTORY-ERROR] Không thể gửi lịch sử phát nhạc tới music-service: %v\n", err)
			return
		}
		defer resp.Body.Close()
	}()
}
