package ws

import (
	"context"
	"errors"
	"testing"

	roomv1 "github.com/worktogether/services/timer-service/api/v1"
	"google.golang.org/grpc"
)

type roomClientStub struct {
	response *roomv1.VerifyRoomMemberResponse
	err      error
}

func (s roomClientStub) VerifyRoomMember(context.Context, *roomv1.VerifyRoomMemberRequest, ...grpc.CallOption) (*roomv1.VerifyRoomMemberResponse, error) {
	return s.response, s.err
}

func TestCanControlTimerAllowsControlPermission(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{
		IsMember:    true,
		Permissions: []string{"CAN_CONTROL_PLAYBACK"},
	}}}

	if !hub.canControlTimer(context.Background(), "room-1", "user-1") {
		t.Fatal("expected a member with CAN_CONTROL_PLAYBACK to control the timer")
	}
}

func TestCanControlTimerRejectsMemberWithoutPermission(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{response: &roomv1.VerifyRoomMemberResponse{
		IsMember:    true,
		Permissions: []string{"CAN_CHAT"},
	}}}

	if hub.canControlTimer(context.Background(), "room-1", "user-1") {
		t.Fatal("expected a member without CAN_CONTROL_PLAYBACK to be rejected")
	}
}

func TestCanControlTimerRejectsRoomServiceFailure(t *testing.T) {
	hub := &Hub{roomClient: roomClientStub{err: errors.New("room service unavailable")}}

	if hub.canControlTimer(context.Background(), "room-1", "user-1") {
		t.Fatal("expected room service failures to deny timer control")
	}
}
