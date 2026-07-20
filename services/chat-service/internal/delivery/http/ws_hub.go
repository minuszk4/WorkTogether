package http

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/worktogether/pkg/translation"
	"github.com/worktogether/services/chat-service/internal/domain"
)

type Client struct {
	UserID         string
	RoomID         string
	Username       string
	Conn           *websocket.Conn
	Send           chan []byte
	Hub            *Hub
	CanChat        bool
	TargetLanguage string
}

func (h *Hub) BroadcastSubtitle(roomID, userID, text, language string) string {
	id := uuid.NewString()
	data, err := json.Marshal(domain.WSMessage{
		Event:  "subtitle:received",
		RoomID: roomID,
		Payload: gin.H{
			"id":        id,
			"sender_id": userID,
			"text":      text,
			"language":  language,
		},
	})
	if err == nil {
		h.BroadcastToRoom(roomID, data)
	}
	return id
}

type MemberState struct {
	IsPlaying  bool  `json:"is_playing"`
	PositionMS int   `json:"position_ms"`
	UpdatedAt  int64 `json:"updated_at"`
}

type RoomPresence struct {
	sync.RWMutex
	MemberStates    map[string]MemberState
	RecentReactions map[string]int
}

type Hub struct {
	sync.RWMutex
	Rooms             map[string]map[*Client]bool
	RoomPresences     map[string]*RoomPresence
	RedisClient       *redis.Client
	translationWorker *translation.Worker
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		Rooms:         make(map[string]map[*Client]bool),
		RoomPresences: make(map[string]*RoomPresence),
		RedisClient:   rdb,
	}
}

func (h *Hub) Register(c *Client) {
	h.Lock()
	defer h.Unlock()
	if h.Rooms[c.RoomID] == nil {
		h.Rooms[c.RoomID] = make(map[*Client]bool)
	}
	h.Rooms[c.RoomID][c] = true
}

func (h *Hub) Unregister(c *Client) {
	h.Lock()
	defer h.Unlock()
	if h.Rooms[c.RoomID] != nil {
		if _, ok := h.Rooms[c.RoomID][c]; ok {
			delete(h.Rooms[c.RoomID], c)
			close(c.Send)
			c.Conn.Close()
			if len(h.Rooms[c.RoomID]) == 0 {
				delete(h.Rooms, c.RoomID)
				// Fix B3: Cleanup RoomPresences to prevent memory leak
				delete(h.RoomPresences, c.RoomID)
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	if h.RedisClient == nil {
		h.localBroadcastToRoom(roomID, message)
		h.queueTranslations(message)
		return
	}
	ctx := context.Background()
	// Parse event type to use correct channel if needed, or default to ch:chat
	h.RedisClient.Publish(ctx, "ch:chat:"+roomID, message)
}

// EnableTranslation starts one bounded worker for this chat-service instance.
// Translation failures are intentionally dropped; chat delivery never waits.
func (h *Hub) EnableTranslation(ctx context.Context, queueSize int, timeout time.Duration, maxCharsPerMinute int, translator translation.Translator) {
	if translator == nil {
		return
	}
	worker := translation.NewWorkerWithRateLimit(queueSize, timeout, maxCharsPerMinute, translator, h.publishTranslation)
	h.Lock()
	h.translationWorker = worker
	h.Unlock()
	go worker.Start(ctx)
}

func (h *Hub) localBroadcastToRoom(roomID string, message []byte) {
	h.RLock()
	defer h.RUnlock()
	if clients, ok := h.Rooms[roomID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				go h.Unregister(client)
			}
		}
	}
}

func (h *Hub) StartRedisSubscriber() {
	pubsub := h.RedisClient.PSubscribe(context.Background(), "ch:chat:*", "ch:presence:*", "ch:reaction:*", "ch:vibe:*")
	go func() {
		for msg := range pubsub.Channel() {
			// e.g. ch:chat:room123
			// Extract roomID
			roomID := ""
			// Find last colon
			for i := len(msg.Channel) - 1; i >= 0; i-- {
				if msg.Channel[i] == ':' {
					roomID = msg.Channel[i+1:]
					break
				}
			}
			if roomID != "" {
				h.localBroadcastToRoom(roomID, []byte(msg.Payload))
				h.queueTranslations([]byte(msg.Payload))
			}
		}
	}()
}

func (h *Hub) queueTranslations(data []byte) {
	var message struct {
		Event   string          `json:"event"`
		RoomID  string          `json:"room_id"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(data, &message) != nil {
		return
	}
	var payload struct {
		ID       string `json:"id"`
		Content  string `json:"content"`
		Text     string `json:"text"`
		SenderID string `json:"sender_id"`
		Language string `json:"language"`
	}
	if json.Unmarshal(message.Payload, &payload) != nil {
		return
	}
	text := payload.Content
	kind := translation.ChatEvent
	if message.Event == "subtitle:received" {
		text = payload.Text
		kind = translation.TranscriptEvent
	}
	if (message.Event != "chat:message_received" && message.Event != "subtitle:received") || message.RoomID == "" || payload.ID == "" || text == "" {
		return
	}
	h.RLock()
	worker := h.translationWorker
	clients := h.Rooms[message.RoomID]
	jobs := make([]translation.Job, 0, len(clients))
	seen := make(map[string]struct{})
	for client := range clients {
		if client.TargetLanguage == "" || client.UserID == payload.SenderID {
			continue
		}
		key := client.UserID + "\x00" + client.TargetLanguage
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		jobs = append(jobs, translation.Job{
			EventID:        payload.ID,
			RoomID:         message.RoomID,
			RecipientID:    client.UserID,
			Kind:           kind,
			Text:           text,
			SourceLanguage: payload.Language,
			TargetLanguage: client.TargetLanguage,
		})
	}
	h.RUnlock()
	if worker == nil {
		return
	}
	for _, job := range jobs {
		worker.Submit(job)
	}
}

func (h *Hub) publishTranslation(result translation.Result) {
	data, err := json.Marshal(domain.WSMessage{
		Event:  "translation:received",
		RoomID: result.RoomID,
		Payload: gin.H{
			"event_id":          result.EventID,
			"kind":              result.Kind,
			"text":              result.Text,
			"target_language":   result.TargetLanguage,
			"detected_language": result.DetectedLanguage,
		},
	})
	if err != nil {
		return
	}
	h.RLock()
	defer h.RUnlock()
	for client := range h.Rooms[result.RoomID] {
		if client.UserID != result.RecipientID || client.TargetLanguage != result.TargetLanguage {
			continue
		}
		select {
		case client.Send <- data:
		default:
			go h.Unregister(client)
		}
	}
}

func (h *Hub) UpdateClientPresence(roomID string, userID string, isPlaying bool, positionMs int) {
	h.Lock()
	presence, exists := h.RoomPresences[roomID]
	if !exists {
		presence = &RoomPresence{
			MemberStates:    make(map[string]MemberState),
			RecentReactions: make(map[string]int),
		}
		h.RoomPresences[roomID] = presence
	}
	h.Unlock()

	presence.Lock()
	presence.MemberStates[userID] = MemberState{
		IsPlaying:  isPlaying,
		PositionMS: positionMs,
		UpdatedAt:  time.Now().UnixMilli(),
	}
	presence.Unlock()
}

func (h *Hub) RecordReaction(roomID string, emoji string) {
	h.Lock()
	presence, exists := h.RoomPresences[roomID]
	if !exists {
		presence = &RoomPresence{
			MemberStates:    make(map[string]MemberState),
			RecentReactions: make(map[string]int),
		}
		h.RoomPresences[roomID] = presence
	}
	h.Unlock()

	presence.Lock()
	presence.RecentReactions[emoji]++
	presence.Unlock()
}

func (h *Hub) determineVibe(reactions map[string]int) (string, map[string]int) {
	hypeCount := reactions["🔥"] + reactions["👏"]
	chillCount := reactions["❤️"] + reactions["😮"]
	studyCount := reactions["📚"]

	currentVibe := "chill" // default
	if hypeCount > chillCount && hypeCount > studyCount {
		currentVibe = "hype"
	} else if studyCount > chillCount && studyCount > hypeCount {
		currentVibe = "study"
	} else if chillCount > 0 {
		currentVibe = "chill"
	}

	scores := map[string]int{
		"chill": chillCount,
		"hype":  hypeCount,
		"study": studyCount,
	}
	return currentVibe, scores
}

func (h *Hub) StartVibeTicker() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			h.Lock()
			activeRooms := make([]string, 0, len(h.RoomPresences))
			for roomID := range h.RoomPresences {
				activeRooms = append(activeRooms, roomID)
			}
			h.Unlock()

			for _, roomID := range activeRooms {
				h.Lock()
				presence, exists := h.RoomPresences[roomID]
				h.Unlock()

				if !exists {
					continue
				}

				presence.Lock()

				currentVibe, scores := h.determineVibe(presence.RecentReactions)

				vibeMsg := domain.WSMessage{
					Event:  "presence:vibe_tick",
					RoomID: roomID,
					Payload: gin.H{
						"current_vibe": currentVibe,
						"vibe_scores": gin.H{
							"chill": scores["chill"],
							"hype":  scores["hype"],
							"study": scores["study"],
						},
					},
				}
				data, _ := json.Marshal(vibeMsg)

				// Decay (clear for next tick)
				presence.RecentReactions = make(map[string]int)
				presence.Unlock()

				go h.BroadcastToRoom(roomID, data)
			}
		}
	}()
}
