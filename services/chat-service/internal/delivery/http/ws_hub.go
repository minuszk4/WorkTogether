package http

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/worktogether/services/chat-service/internal/domain"
)

type Client struct {
	UserID   string
	RoomID   string
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	CanChat  bool
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
	Rooms         map[string]map[*Client]bool
	RoomPresences map[string]*RoomPresence
}

func NewHub() *Hub {
	return &Hub{
		Rooms:         make(map[string]map[*Client]bool),
		RoomPresences: make(map[string]*RoomPresence),
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
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
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
			for roomID, presence := range h.RoomPresences {
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
			h.Unlock()
		}
	}()
}
