package http

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/worktogether/pkg/translation"
	"github.com/worktogether/services/chat-service/internal/domain"
)

func TestNewHub(t *testing.T) {
	h := NewHub(nil)
	if h.RoomPresences == nil {
		t.Error("Expected RoomPresences map to be initialized, got nil")
	}
}

func TestUpdateClientPresence(t *testing.T) {
	h := NewHub(nil)
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

func TestRecordReaction(t *testing.T) {
	h := NewHub(nil)
	h.RecordReaction("room-1", "🔥")

	presence, exists := h.RoomPresences["room-1"]
	if !exists {
		t.Fatal("Expected RoomPresence to be created")
	}

	count, exists := presence.RecentReactions["🔥"]
	if !exists {
		t.Fatal("Expected reaction count to be recorded")
	}

	if count != 1 {
		t.Errorf("Expected reaction count to be 1, got %d", count)
	}
}

func TestDetermineVibe(t *testing.T) {
	h := NewHub(nil)

	// Case 1: Hype dominates
	r1 := map[string]int{"🔥": 5, "❤️": 2}
	vibe, _ := h.determineVibe(r1)
	if vibe != "hype" {
		t.Errorf("Expected hype, got %s", vibe)
	}

	// Case 2: Study dominates
	r2 := map[string]int{"📚": 3, "👏": 1}
	vibe, _ = h.determineVibe(r2)
	if vibe != "study" {
		t.Errorf("Expected study, got %s", vibe)
	}

	// Case 3: Chill dominates
	r3 := map[string]int{"😮": 4, "📚": 1}
	vibe, _ = h.determineVibe(r3)
	if vibe != "chill" {
		t.Errorf("Expected chill, got %s", vibe)
	}
}

func TestHubDeliversTranslationOnlyToTheConfiguredRecipient(t *testing.T) {
	hub := NewHub(nil)
	hub.EnableTranslation(context.Background(), 2, time.Second, 0, func(_ context.Context, job translation.Job) (translation.Result, error) {
		return translation.Result{Job: job, Text: "xin chào"}, nil
	})
	client := &Client{UserID: "user-1", RoomID: "room-1", TargetLanguage: "vi", Send: make(chan []byte, 1)}
	hub.Register(client)

	original, err := json.Marshal(domain.WSMessage{
		Event:   "chat:message_received",
		RoomID:  "room-1",
		Payload: map[string]any{"id": "message-1", "content": "hello", "sender_id": "user-2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hub.queueTranslations(original)

	select {
	case data := <-client.Send:
		var message domain.WSMessage
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatal(err)
		}
		if message.Event != "translation:received" {
			t.Fatalf("event = %q", message.Event)
		}
		payload := message.Payload.(map[string]any)
		if payload["event_id"] != "message-1" || payload["text"] != "xin chào" {
			t.Fatalf("unexpected translation payload: %#v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("translation was not delivered")
	}
}
