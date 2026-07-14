package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	roomv1 "github.com/worktogether/services/collab-service/api/v1"
	"google.golang.org/grpc"
)

type roomClientStub struct {
	response *roomv1.VerifyRoomMemberResponse
	err      error
}

func (s roomClientStub) VerifyRoomMember(context.Context, *roomv1.VerifyRoomMemberRequest, ...grpc.CallOption) (*roomv1.VerifyRoomMemberResponse, error) {
	return s.response, s.err
}

func TestCanAccessRoomRejectsNonMember(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{IsMember: false}}}
	if hub.canAccessRoom(context.Background(), "room-1", "user-1") {
		t.Fatal("expected a non-member to be rejected")
	}
}

func TestCanAccessRoomAllowsMember(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{IsMember: true}}}
	if !hub.canAccessRoom(context.Background(), "room-1", "user-1") {
		t.Fatal("expected a member to be allowed")
	}
}

func TestCanAccessRoomRejectsRoomServiceFailure(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{err: errors.New("room service unavailable")}}
	if hub.canAccessRoom(context.Background(), "room-1", "user-1") {
		t.Fatal("expected a room service failure to deny access")
	}
}

func TestServeCollabWSRejectsNonMemberBeforeUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := &Hub{jwtSecret: "secret", roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{IsMember: false}}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-1", "type": "access_token"}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/api/v1/rooms/:room_id/collab/ws", ServeCollabWS(hub))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/room-2/collab/ws?token="+token, nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected non-member to be rejected before upgrade, got %d", recorder.Code)
	}
}
