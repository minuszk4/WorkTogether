package http

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/worktogether/services/auth-service/internal/domain"
	"github.com/worktogether/services/auth-service/internal/usecase"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	usecase      *usecase.AuthUsecase
	googleCfg    *oauth2.Config
	frontendURL  string
	cookieSecure bool
}

func NewAuthHandler(uc *usecase.AuthUsecase, googleCfg *oauth2.Config, frontendURL string) *AuthHandler {
	return &AuthHandler{
		usecase:      uc,
		googleCfg:    googleCfg,
		frontendURL:  frontendURL,
		cookieSecure: strings.HasPrefix(frontendURL, "https://"),
	}
}

// ─── Register ────────────────────────────────────────────────────────────────

func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_PARAMETERS", err.Error()))
		return
	}

	_, err := h.usecase.Register(c.Request.Context(), &req)
	if err != nil {
		code := "REGISTRATION_FAILED"
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrEmailAlreadyExists) || errors.Is(err, usecase.ErrUsernameExists) {
			code = "ACCOUNT_EXISTS"
			status = http.StatusConflict
		}
		c.JSON(status, errorResp(code, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, successResp(gin.H{
		"message": "Đăng ký thành công. Vui lòng kiểm tra email để xác thực tài khoản.",
	}))
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		var req domain.VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			token = req.Token
		}
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_PARAMETERS", "Thiếu token xác thực."))
		return
	}

	err := h.usecase.VerifyEmail(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_TOKEN", err.Error()))
		return
	}

	// Redirect về frontend với thông báo thành công
	c.Redirect(http.StatusFound, h.frontendURL+"/auth?verified=true")
}

// ─── Login ────────────────────────────────────────────────────────────────────

func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("INVALID_PARAMETERS", err.Error()))
		return
	}

	acc, accessToken, refreshToken, err := h.usecase.Login(c.Request.Context(), &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		status := http.StatusUnauthorized
		code := "AUTH_ERROR"
		if errors.Is(err, usecase.ErrAccountNotVerified) {
			status = http.StatusForbidden
			code = "EMAIL_NOT_VERIFIED"
		}
		c.JSON(status, errorResp(code, err.Error()))
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/api/v1/auth", "", h.cookieSecure, true)

	c.JSON(http.StatusOK, successResp(gin.H{
		"access_token": accessToken,
		"user_id":      acc.ID,
		"username":     acc.Username,
		"expires_in":   900,
	}))
}

// ─── Google OAuth ─────────────────────────────────────────────────────────────

// GoogleLogin redirect người dùng sang trang đăng nhập Google
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	if !usecase.IsGoogleConfigured() {
		c.JSON(http.StatusServiceUnavailable, errorResp("GOOGLE_NOT_CONFIGURED",
			"Đăng nhập Google chưa được cấu hình. Vui lòng thêm GOOGLE_CLIENT_ID và GOOGLE_CLIENT_SECRET vào .env"))
		return
	}

	state := generateState()
	// Lưu state vào cookie để verify ở callback
	c.SetCookie("oauth_state", state, 600, "/", "", h.cookieSecure, true)
	url := usecase.GetGoogleAuthURL(h.googleCfg, state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback xử lý callback sau khi người dùng xác thực với Google
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Verify state chống CSRF
	stateCookie, _ := c.Cookie("oauth_state")
	stateParam := c.Query("state")
	if stateCookie == "" || stateCookie != stateParam {
		c.Redirect(http.StatusFound, h.frontendURL+"/auth?error=invalid_state")
		return
	}

	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, h.frontendURL+"/auth?error=no_code")
		return
	}

	// Exchange code lấy user info từ Google
	googleUser, err := usecase.ExchangeGoogleCode(c.Request.Context(), h.googleCfg, code)
	if err != nil {
		c.Redirect(http.StatusFound, h.frontendURL+"/auth?error=google_exchange_failed")
		return
	}

	// Tạo/tìm tài khoản và sinh JWT
	acc, accessToken, refreshToken, err := h.usecase.LoginWithGoogle(
		c.Request.Context(), googleUser, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		c.Redirect(http.StatusFound, h.frontendURL+"/auth?error=login_failed")
		return
	}

	// Xóa state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", h.cookieSecure, true)
	// Set refresh token cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/api/v1/auth", "", h.cookieSecure, true)

	// Redirect về frontend kèm access token (fragment để không lưu vào server log)
	c.Redirect(http.StatusFound,
		h.frontendURL+"/auth/callback#access_token="+accessToken+"&user_id="+acc.ID+"&username="+acc.Username)
}

// ─── Refresh & Logout ─────────────────────────────────────────────────────────

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("MISSING_COOKIE", "Không tìm thấy token làm mới."))
		return
	}

	accessToken, newRefreshToken, err := h.usecase.RefreshToken(c.Request.Context(), refreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResp("REFRESH_FAILED", err.Error()))
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", newRefreshToken, 7*24*3600, "/api/v1/auth", "", h.cookieSecure, true)
	c.JSON(http.StatusOK, successResp(gin.H{"access_token": accessToken, "expires_in": 900}))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		_ = h.usecase.Logout(c.Request.Context(), refreshToken)
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth", "", h.cookieSecure, true)
	c.JSON(http.StatusOK, successResp(gin.H{"message": "Đăng xuất thành công."}))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func successResp(data interface{}) gin.H {
	return gin.H{"success": true, "data": data, "error": nil}
}

func errorResp(code, message string) gin.H {
	return gin.H{"success": false, "data": nil, "error": gin.H{"code": code, "message": message}}
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
