package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/user-service/internal/domain"
)

type statusUsecaseFake struct {
	save      *domain.UpdateStatusRequest
	heartbeat *domain.HeartbeatRequest
}

func (f *statusUsecaseFake) UpdatePresence(_ context.Context, _ string, req *domain.UpdateStatusRequest) (*domain.Presence, error) {
	f.save = req
	return &domain.Presence{Status: req.Status, CustomText: req.CustomText}, nil
}

func (f *statusUsecaseFake) HeartbeatPresence(_ context.Context, _ string, req *domain.HeartbeatRequest) (*domain.Presence, error) {
	f.heartbeat = req
	return &domain.Presence{Status: req.Status}, nil
}

func statusRequest(t *testing.T, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.PUT("/status", func(c *gin.Context) { c.Set("userID", "user-1"); handler(c) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/status", bytes.NewBufferString(body)))
	return w
}

func TestHeartbeatDoesNotWriteCustomText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &statusUsecaseFake{}
	h := &UserHandler{statusUsecase: uc}

	w := statusRequest(t, h.UpdatePresence, `{"status":"online","custom_text":"studying"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", w.Code, w.Body.String())
	}
	w = statusRequest(t, h.HeartbeatPresence, `{"status":"away"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if uc.heartbeat == nil || uc.heartbeat.Status != "away" {
		t.Fatalf("heartbeat = %#v, want away", uc.heartbeat)
	}
	if uc.save == nil || uc.save.CustomText != "studying" {
		t.Fatalf("save request = %#v, want persisted custom text", uc.save)
	}
}

func TestUpdateStatusRejectsCustomTextOver100Characters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &UserHandler{statusUsecase: &statusUsecaseFake{}}

	w := statusRequest(t, h.UpdatePresence, `{"status":"online","custom_text":"`+strings.Repeat("a", 101)+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
