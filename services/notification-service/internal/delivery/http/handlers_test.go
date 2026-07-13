package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAdminRejectsRegularMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("isAdmin", false)

	handler := &NotificationHandler{}
	if handler.requireAdmin(context) {
		t.Fatal("expected a regular member to be rejected")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

func TestSSETokenSubjectRejectsMissingSubject(t *testing.T) {
	secret := "test-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"type": "access_token"})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	if _, err := sseTokenSubject(tokenString, secret); err == nil {
		t.Fatal("expected access token without sub to be rejected")
	}
}
