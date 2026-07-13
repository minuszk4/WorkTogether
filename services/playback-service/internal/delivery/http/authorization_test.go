package http

import (
	"context"
	"testing"

	roomv1 "github.com/worktogether/services/playback-service/api/v1"
	"google.golang.org/grpc"
)

type roomClientStub struct {
	response *roomv1.VerifyRoomMemberResponse
}

func (s roomClientStub) VerifyRoomMember(context.Context, *roomv1.VerifyRoomMemberRequest, ...grpc.CallOption) (*roomv1.VerifyRoomMemberResponse, error) {
	return s.response, nil
}

func TestCanControlPlaybackRequiresRoomPermission(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{IsMember: true, Permissions: []string{"CAN_CONTROL_PLAYBACK"}}}}
	if !hub.canControlPlayback(context.Background(), "room-1", "user-1") {
		t.Fatal("expected CAN_CONTROL_PLAYBACK to allow playback control")
	}
}

func TestCanControlPlaybackRejectsMemberWithoutPermission(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{IsMember: true, Permissions: []string{"CAN_CHAT"}}}}
	if hub.canControlPlayback(context.Background(), "room-1", "user-1") {
		t.Fatal("expected member without CAN_CONTROL_PLAYBACK to be rejected")
	}
}
