package http

import "testing"

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h.RoomPresences == nil {
		t.Error("Expected RoomPresences map to be initialized, got nil")
	}
}
