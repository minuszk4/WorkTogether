package http

import "testing"

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h.RoomPresences == nil {
		t.Error("Expected RoomPresences map to be initialized, got nil")
	}
}

func TestUpdateClientPresence(t *testing.T) {
	h := NewHub()
	h.UpdateClientPresence("room-1", "user-1", true, 5000)

	presence, exists := h.RoomPresences["room-1"]
	if !exists {
		t.Fatal("Expected RoomPresence to be created")
	}

	state, exists := presence.MemberStates["user-1"]
	if !exists {
		t.Fatal("Expected member state for user-1 to be saved")
	}

	if !state.IsPlaying || state.PositionMS != 5000 {
		t.Errorf("Unexpected member state saved: %+v", state)
	}
}
