package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	roomv1 "github.com/worktogether/services/voice-service/api/v1"
	"github.com/worktogether/services/voice-service/internal/usecase"
	"google.golang.org/grpc"
)

type roomClientStub struct {
	response *roomv1.VerifyRoomMemberResponse
}

func (s roomClientStub) VerifyRoomMember(context.Context, *roomv1.VerifyRoomMemberRequest, ...grpc.CallOption) (*roomv1.VerifyRoomMemberResponse, error) {
	return s.response, nil
}

func TestGetTokenRequiresVoicePermission(t *testing.T) {
	response := requestVoiceToken(t, roomClientStub{response: &roomv1.VerifyRoomMemberResponse{
		IsMember: true,
	}}, `{"publish_sources":["microphone"]}`)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestGetTokenRejectsDifferentActiveBreakout(t *testing.T) {
	response := requestVoiceToken(t, roomClientStub{response: &roomv1.VerifyRoomMemberResponse{
		IsMember:        true,
		Permissions:     []string{"CAN_USE_VOICE"},
		ActiveSubRoomID: "sub-allowed",
	}}, `{"channel_id":"sub-other","publish_sources":["microphone"]}`)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestGetTokenReturnsServerGeneratedBreakoutRoomName(t *testing.T) {
	response := requestVoiceToken(t, roomClientStub{response: &roomv1.VerifyRoomMemberResponse{
		IsMember:        true,
		Permissions:     []string{"CAN_USE_VOICE"},
		ActiveSubRoomID: "sub-allowed",
	}}, `{"channel_id":"sub-allowed","publish_sources":["microphone"]}`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var body struct {
		Data struct {
			RoomName  string `json:"room_name"`
			ExpiresIn int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.RoomName != "room:room-123:sub:sub-allowed" || body.Data.ExpiresIn != 600 {
		t.Fatalf("unexpected token response: %#v", body.Data)
	}
}

func requestVoiceToken(t *testing.T, roomClient roomv1.RoomInternalServiceClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewVoiceHandler(
		usecase.NewVoiceUsecase("wss://livekit.example.com", "test-key", "test-secret", nil),
		roomClient,
		"wss://livekit.example.com",
		"test-key",
		"test-secret",
	)
	router := gin.New()
	router.POST("/voice/rooms/:room_id/token", func(c *gin.Context) {
		c.Set("userID", "user-12345678")
		handler.GetToken(c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/voice/rooms/room-123/token", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}
