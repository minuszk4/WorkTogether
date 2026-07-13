package domain

import "time"

type Account struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsVerified   bool      `json:"is_verified"`
	IsAdmin      bool      `json:"is_admin"`
	IsSuspended  bool      `json:"is_suspended"`
	GoogleID     string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"account_id"`
	RefreshToken string    `json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Identity string `json:"identity" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}
