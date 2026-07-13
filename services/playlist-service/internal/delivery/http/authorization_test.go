package http

import "testing"

func TestHasPermissionMatchesCaseInsensitively(t *testing.T) {
	if !hasPermission([]string{"can_manage_playlist"}, "CAN_MANAGE_PLAYLIST") {
		t.Fatal("expected matching permission to be accepted")
	}
}

func TestHasPermissionRejectsMissingPermission(t *testing.T) {
	if hasPermission([]string{"CAN_CHAT"}, "CAN_MANAGE_PLAYLIST") {
		t.Fatal("expected missing permission to be rejected")
	}
}
