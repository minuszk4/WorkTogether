package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/worktogether/services/auth-service/internal/domain"
	"github.com/worktogether/services/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("tài khoản hoặc mật khẩu không chính xác")
	ErrAccountNotVerified = errors.New("tài khoản chưa được xác thực email")
	ErrEmailAlreadyExists = errors.New("email đã tồn tại trên hệ thống")
	ErrUsernameExists     = errors.New("tên tài khoản đã tồn tại")
	ErrSessionExpired     = errors.New("phiên làm việc đã hết hạn, vui lòng đăng nhập lại")
	ErrInvalidSession     = errors.New("phiên làm việc không hợp lệ")
	ErrInvalidVerifyToken = errors.New("mã xác thực không hợp lệ hoặc đã hết hạn")
	ErrGoogleNotConfigured = errors.New("đăng nhập Google chưa được cấu hình")
)

type AuthUsecase struct {
	repo         *repository.PostgresRepository
	emailSvc     *EmailService
	jwtSecret    []byte
	jwtExpMins   int
	appBaseURL   string
}

func NewAuthUsecase(repo *repository.PostgresRepository, emailSvc *EmailService, secret string, jwtExpMins int) *AuthUsecase {
	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:8080"
	}
	return &AuthUsecase{
		repo:       repo,
		emailSvc:   emailSvc,
		jwtSecret:  []byte(secret),
		jwtExpMins: jwtExpMins,
		appBaseURL: appBaseURL,
	}
}

// ─── Register ────────────────────────────────────────────────────────────────

func (u *AuthUsecase) Register(ctx context.Context, req *domain.RegisterRequest) (string, error) {
	// Kiểm tra trùng email/username
	existing, _ := u.repo.GetAccountByIdentity(ctx, req.Email)
	if existing != nil {
		return "", ErrEmailAlreadyExists
	}
	existing, _ = u.repo.GetAccountByIdentity(ctx, req.Username)
	if existing != nil {
		return "", ErrUsernameExists
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Trong dev mode (AUTO_VERIFY_EMAIL=true), tài khoản tự động được xác thực
	autoVerify := os.Getenv("AUTO_VERIFY_EMAIL") == "true"

	acc := &domain.Account{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashed),
		IsVerified:   autoVerify,
	}

	if err := u.repo.CreateAccount(ctx, acc); err != nil {
		return "", err
	}

	if autoVerify {
		fmt.Printf("[DEV] Tài khoản %s tự động xác thực\n", acc.Email)
		// Gửi email chào mừng
		go u.emailSvc.SendWelcomeEmail(acc.Email, acc.Username)
		return "dev-auto-verified", nil
	}

	// Sinh JWT token xác thực email (24h)
	verifyClaims := jwt.MapClaims{
		"sub":   acc.ID,
		"email": acc.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"type":  "email_verification",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, verifyClaims)
	tokenStr, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return "", err
	}

	verifyURL := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", u.appBaseURL, tokenStr)

	// Gửi email bất đồng bộ
	go func() {
		if err := u.emailSvc.SendVerificationEmail(acc.Email, acc.Username, verifyURL); err != nil {
			fmt.Printf("[EMAIL-ERROR] Không thể gửi email xác thực tới %s: %v\n", acc.Email, err)
		}
	}()

	return tokenStr, nil
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func (u *AuthUsecase) VerifyEmail(ctx context.Context, tokenStr string) error {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return u.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return ErrInvalidVerifyToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "email_verification" {
		return ErrInvalidVerifyToken
	}

	accountID, ok := claims["sub"].(string)
	if !ok {
		return ErrInvalidVerifyToken
	}

	if err := u.repo.UpdateAccountVerification(ctx, accountID, true); err != nil {
		return err
	}

	// Gửi email chào mừng sau khi xác thực
	acc, _ := u.repo.GetAccountByID(ctx, accountID)
	if acc != nil {
		go u.emailSvc.SendWelcomeEmail(acc.Email, acc.Username)
	}

	return nil
}

// ─── Login (email/password) ───────────────────────────────────────────────────

func (u *AuthUsecase) Login(ctx context.Context, req *domain.LoginRequest, ip, ua string) (*domain.Account, string, string, error) {
	acc, err := u.repo.GetAccountByIdentity(ctx, req.Identity)
	if err != nil {
		return nil, "", "", err
	}
	if acc == nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Tài khoản Google không có password
	if acc.PasswordHash == "" {
		return nil, "", "", errors.New("tài khoản này đăng ký qua Google, vui lòng đăng nhập bằng Google")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	if !acc.IsVerified {
		return nil, "", "", ErrAccountNotVerified
	}

	return u.createSession(ctx, acc, ip, ua)
}

// ─── Google OAuth ─────────────────────────────────────────────────────────────

// LoginWithGoogle xử lý callback từ Google, tạo/tìm tài khoản, trả về tokens
func (u *AuthUsecase) LoginWithGoogle(ctx context.Context, googleUser *GoogleUserInfo, ip, ua string) (*domain.Account, string, string, error) {
	// 1. Tìm theo google_id
	acc, err := u.repo.GetAccountByGoogleID(ctx, googleUser.ID)
	if err != nil {
		return nil, "", "", err
	}

	// 2. Nếu chưa có, tìm theo email (có thể đã đăng ký bằng email trước)
	if acc == nil {
		acc, err = u.repo.GetAccountByEmail(ctx, googleUser.Email)
		if err != nil {
			return nil, "", "", err
		}
		// Link google_id vào tài khoản cũ
		if acc != nil {
			_ = u.repo.LinkGoogleID(ctx, acc.ID, googleUser.ID)
			acc.GoogleID = googleUser.ID
		}
	}

	// 3. Tạo tài khoản mới nếu chưa tồn tại
	if acc == nil {
		username := u.generateUsernameFromEmail(ctx, googleUser.Email, googleUser.Name)
		acc = &domain.Account{
			Email:      googleUser.Email,
			Username:   username,
			IsVerified: true, // Google đã xác thực email
			GoogleID:   googleUser.ID,
		}
		if err := u.repo.CreateAccount(ctx, acc); err != nil {
			return nil, "", "", fmt.Errorf("tạo tài khoản Google thất bại: %w", err)
		}
		// Gửi email chào mừng
		go u.emailSvc.SendWelcomeEmail(acc.Email, acc.Username)
	}

	return u.createSession(ctx, acc, ip, ua)
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

func (u *AuthUsecase) RefreshToken(ctx context.Context, tokenStr, ip, ua string) (string, string, error) {
	sess, err := u.repo.GetSessionByToken(ctx, tokenStr)
	if err != nil {
		return "", "", err
	}
	if sess == nil {
		return "", "", ErrInvalidSession
	}

	if time.Now().After(sess.ExpiresAt) {
		_ = u.repo.DeleteSession(ctx, tokenStr)
		return "", "", ErrSessionExpired
	}

	acc, err := u.repo.GetAccountByID(ctx, sess.AccountID)
	if err != nil || acc == nil {
		return "", "", ErrInvalidSession
	}

	accessToken, err := u.generateAccessToken(acc)
	if err != nil {
		return "", "", err
	}

	newRefreshToken := uuid.New().String()
	newSess := &domain.Session{
		AccountID:    acc.ID,
		RefreshToken: newRefreshToken,
		IPAddress:    ip,
		UserAgent:    ua,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	_ = u.repo.DeleteSession(ctx, tokenStr)
	if err := u.repo.CreateSession(ctx, newSess); err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func (u *AuthUsecase) Logout(ctx context.Context, tokenStr string) error {
	return u.repo.DeleteSession(ctx, tokenStr)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (u *AuthUsecase) createSession(ctx context.Context, acc *domain.Account, ip, ua string) (*domain.Account, string, string, error) {
	accessToken, err := u.generateAccessToken(acc)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken := uuid.New().String()
	sess := &domain.Session{
		AccountID:    acc.ID,
		RefreshToken: refreshToken,
		IPAddress:    ip,
		UserAgent:    ua,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	if err := u.repo.CreateSession(ctx, sess); err != nil {
		return nil, "", "", err
	}

	return acc, accessToken, refreshToken, nil
}

func (u *AuthUsecase) generateAccessToken(acc *domain.Account) (string, error) {
	claims := jwt.MapClaims{
		"sub":      acc.ID,
		"username": acc.Username,
		"exp":      time.Now().Add(time.Duration(u.jwtExpMins) * time.Minute).Unix(),
		"type":     "access_token",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(u.jwtSecret)
}

// generateUsernameFromEmail sinh username từ email, đảm bảo không trùng
func (u *AuthUsecase) generateUsernameFromEmail(ctx context.Context, email, displayName string) string {
	base := strings.Split(email, "@")[0]
	// Sanitize: chỉ giữ chữ thường, số, dấu _
	var sanitized strings.Builder
	for _, c := range strings.ToLower(base) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			sanitized.WriteRune(c)
		}
	}
	candidate := sanitized.String()
	if len(candidate) < 3 {
		candidate = "user_" + candidate
	}

	// Kiểm tra trùng và thêm suffix nếu cần
	for i := 0; i < 10; i++ {
		name := candidate
		if i > 0 {
			name = fmt.Sprintf("%s_%d", candidate, i)
		}
		existing, _ := u.repo.GetAccountByIdentity(ctx, name)
		if existing == nil {
			return name
		}
	}
	return fmt.Sprintf("%s_%s", candidate, uuid.New().String()[:6])
}

func (u *AuthUsecase) ForgotPassword(ctx context.Context, email string) error {
	acc, err := u.repo.GetAccountByEmail(ctx, email)
	if err != nil || acc == nil {
		return errors.New("không tìm thấy tài khoản với email này")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  acc.ID,
		"type": "password_reset",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	})
	tokenStr, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return err
	}
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", u.appBaseURL, tokenStr)
	return u.emailSvc.SendPasswordResetEmail(acc.Email, acc.Username, resetURL)
}

func (u *AuthUsecase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return u.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return errors.New("token không hợp lệ hoặc đã hết hạn")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "password_reset" {
		return errors.New("token không hợp lệ")
	}
	accountID, _ := claims["sub"].(string)
	hashedPassword, err := u.hashPassword(newPassword)
	if err != nil {
		return err
	}
	return u.repo.UpdatePassword(ctx, accountID, hashedPassword)
}

func (u *AuthUsecase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	acc, err := u.repo.GetAccountByID(ctx, userID)
	if err != nil || acc == nil {
		return errors.New("tài khoản không tồn tại")
	}
	if !u.checkPasswordHash(oldPassword, acc.PasswordHash) {
		return errors.New("mật khẩu cũ không chính xác")
	}
	hashedPassword, err := u.hashPassword(newPassword)
	if err != nil {
		return err
	}
	return u.repo.UpdatePassword(ctx, userID, hashedPassword)
}

func (u *AuthUsecase) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (u *AuthUsecase) checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

