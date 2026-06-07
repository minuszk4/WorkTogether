package http

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID   string
	RoomID   string
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

type Hub struct {
	sync.RWMutex
	Rooms map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]map[*Client]bool),
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
