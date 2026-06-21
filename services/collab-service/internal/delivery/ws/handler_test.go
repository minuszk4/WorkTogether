package ws

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalWSEvents(t *testing.T) {
	// Test block:add payload parsing
	rawJson := `{"event": "block:add", "room_id": "room123", "payload": "{\"content\": \"test content\", \"block_type\": \"todo\"}"}`
	var msg WSMessage
	err := json.Unmarshal([]byte(rawJson), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal WSMessage: %v", err)
	}

	if msg.Event != "block:add" {
		t.Errorf("Expected event to be 'block:add', got '%s'", msg.Event)
	}
	if msg.RoomID != "room123" {
		t.Errorf("Expected room_id to be 'room123', got '%s'", msg.RoomID)
	}

	// Payload is encoded as a string within raw JSON, so we unquote/unmarshal it
	var innerJson string
	err = json.Unmarshal(msg.Payload, &innerJson)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload string: %v", err)
	}

	var payload struct {
		Content   string `json:"content"`
		BlockType string `json:"block_type"`
	}
	err = json.Unmarshal([]byte(innerJson), &payload)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload content: %v", err)
	}

	if payload.Content != "test content" {
		t.Errorf("Expected content to be 'test content', got '%s'", payload.Content)
	}
	if payload.BlockType != "todo" {
		t.Errorf("Expected block_type to be 'todo', got '%s'", payload.BlockType)
	}
}
