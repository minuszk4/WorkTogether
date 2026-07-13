package usecase

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/worktogether/services/auth-service/internal/domain"
	"github.com/worktogether/services/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func TestGenerateAccessTokenIncludesAdminClaim(t *testing.T) {
	uc := NewAuthUsecase(nil, NewEmailService(), "test_secret", 15)
	tokenStr, err := uc.generateAccessToken(&domain.Account{ID: "admin-1", Username: "admin", IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) { return []byte("test_secret"), nil })
	if err != nil || !token.Valid {
		t.Fatalf("invalid token: %v", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if isAdmin, ok := claims["is_admin"].(bool); !ok || !isAdmin {
		t.Fatalf("expected is_admin claim, got %#v", claims["is_admin"])
	}
}

func TestSetAccountAdminRequiresAdministrator(t *testing.T) {
	uc := NewAuthUsecase(nil, NewEmailService(), "test_secret", 15)
	if err := uc.SetAccountAdmin(context.Background(), false, "actor", "target", true); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestSetAccountSuspendedRequiresAdministrator(t *testing.T) {
	uc := NewAuthUsecase(nil, NewEmailService(), "test_secret", 15)
	if err := uc.SetAccountSuspended(context.Background(), false, "actor", "target", true); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLoginRejectsSuspendedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	uc := NewAuthUsecase(repository.NewPostgresRepository(db), NewEmailService(), "test_secret", 15)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE email = $1 OR username = $2")).
		WithArgs("suspended", "suspended").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "is_verified", "is_admin", "google_id", "created_at", "updated_at"}).
			AddRow("account-1", "suspended@example.com", "suspended", string(passwordHash), true, false, "", time.Now(), time.Now()))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(is_suspended,false) FROM accounts WHERE id = $1")).
		WithArgs("account-1").
		WillReturnRows(sqlmock.NewRows([]string{"is_suspended"}).AddRow(true))

	_, _, _, err = uc.Login(context.Background(), &domain.LoginRequest{Identity: "suspended", Password: "correct-password"}, "127.0.0.1", "test")
	if err != ErrAccountSuspended {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthUsecase_ForgotPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	emailSvc := NewEmailService() // not configured, will mock print
	jwtSecret := "test_secret"
	jwtExpMins := 15
	uc := NewAuthUsecase(repo, emailSvc, jwtSecret, jwtExpMins)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "test@example.com"
		expectedQuery := regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE email = $1")

		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "is_verified", "is_admin", "google_id", "created_at", "updated_at"}).
			AddRow("acc-123", email, "testuser", "some_hash", true, false, "", time.Now(), time.Now())

		mock.ExpectQuery(expectedQuery).WithArgs(email).WillReturnRows(rows)

		err := uc.ForgotPassword(ctx, email)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})

	t.Run("account not found", func(t *testing.T) {
		email := "unknown@example.com"
		expectedQuery := regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE email = $1")

		mock.ExpectQuery(expectedQuery).WithArgs(email).WillReturnError(sql.ErrNoRows)

		err := uc.ForgotPassword(ctx, email)
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "không tìm thấy tài khoản với email này" {
			t.Errorf("unexpected error message: %s", err.Error())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})
}

func TestAuthUsecase_ResetPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	emailSvc := NewEmailService()
	jwtSecret := "test_secret"
	jwtExpMins := 15
	uc := NewAuthUsecase(repo, emailSvc, jwtSecret, jwtExpMins)

	ctx := context.Background()

	// Generate valid reset token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "acc-123",
		"type": "password_reset",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	t.Run("success", func(t *testing.T) {
		newPassword := "newpassword123"
		expectedUpdateQuery := regexp.QuoteMeta("UPDATE accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2")
		expectedDeleteSessionsQuery := regexp.QuoteMeta("DELETE FROM sessions WHERE account_id = $1")

		mock.ExpectExec(expectedUpdateQuery).
			WithArgs(sqlmock.AnyArg(), "acc-123").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(expectedDeleteSessionsQuery).
			WithArgs("acc-123").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := uc.ResetPassword(ctx, tokenStr, newPassword)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})

	t.Run("account not found", func(t *testing.T) {
		newPassword := "newpassword123"
		expectedUpdateQuery := regexp.QuoteMeta("UPDATE accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2")

		mock.ExpectExec(expectedUpdateQuery).
			WithArgs(sqlmock.AnyArg(), "acc-123").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := uc.ResetPassword(ctx, tokenStr, newPassword)
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "tài khoản không tồn tại" {
			t.Errorf("unexpected error: %s", err.Error())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})

	t.Run("invalid token type", func(t *testing.T) {
		invalidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "acc-123",
			"type": "access_token",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})
		invalidTokenStr, _ := invalidToken.SignedString([]byte(jwtSecret))

		err := uc.ResetPassword(ctx, invalidTokenStr, "newpassword123")
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "token không hợp lệ" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	})

	t.Run("missing sub claim", func(t *testing.T) {
		missingSubToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"type": "password_reset",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})
		missingSubTokenStr, _ := missingSubToken.SignedString([]byte(jwtSecret))

		err := uc.ResetPassword(ctx, missingSubTokenStr, "newpassword123")
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "token không hợp lệ" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	})

	t.Run("empty sub claim", func(t *testing.T) {
		emptySubToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "",
			"type": "password_reset",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})
		emptySubTokenStr, _ := emptySubToken.SignedString([]byte(jwtSecret))

		err := uc.ResetPassword(ctx, emptySubTokenStr, "newpassword123")
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "token không hợp lệ" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	})

	t.Run("invalid sub type", func(t *testing.T) {
		invalidSubTypeToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  12345,
			"type": "password_reset",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})
		invalidSubTypeTokenStr, _ := invalidSubTypeToken.SignedString([]byte(jwtSecret))

		err := uc.ResetPassword(ctx, invalidSubTypeTokenStr, "newpassword123")
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "token không hợp lệ" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	})

	t.Run("invalid secret key", func(t *testing.T) {
		wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "acc-123",
			"type": "password_reset",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})
		wrongTokenStr, _ := wrongToken.SignedString([]byte("wrong_secret"))

		err := uc.ResetPassword(ctx, wrongTokenStr, "newpassword123")
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "token không hợp lệ hoặc đã hết hạn" {
			t.Errorf("unexpected error: %s", err.Error())
		}
	})
}

func TestAuthUsecase_ChangePassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %s", err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)
	emailSvc := NewEmailService()
	jwtSecret := "test_secret"
	jwtExpMins := 15
	uc := NewAuthUsecase(repo, emailSvc, jwtSecret, jwtExpMins)

	ctx := context.Background()
	userID := "acc-123"
	oldPassword := "oldpassword123"
	newPassword := "newpassword123"

	hashedOldPassword, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	t.Run("success", func(t *testing.T) {
		expectedGetQuery := regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE id = $1")
		expectedUpdateQuery := regexp.QuoteMeta("UPDATE accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2")
		expectedDeleteSessionsQuery := regexp.QuoteMeta("DELETE FROM sessions WHERE account_id = $1")

		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "is_verified", "is_admin", "google_id", "created_at", "updated_at"}).
			AddRow(userID, "test@example.com", "testuser", string(hashedOldPassword), true, false, "", time.Now(), time.Now())

		mock.ExpectQuery(expectedGetQuery).WithArgs(userID).WillReturnRows(rows)
		mock.ExpectExec(expectedUpdateQuery).WithArgs(sqlmock.AnyArg(), userID).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(expectedDeleteSessionsQuery).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		err := uc.ChangePassword(ctx, userID, oldPassword, newPassword)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})

	t.Run("incorrect old password", func(t *testing.T) {
		expectedGetQuery := regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE id = $1")

		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "is_verified", "is_admin", "google_id", "created_at", "updated_at"}).
			AddRow(userID, "test@example.com", "testuser", string(hashedOldPassword), true, false, "", time.Now(), time.Now())

		mock.ExpectQuery(expectedGetQuery).WithArgs(userID).WillReturnRows(rows)

		err := uc.ChangePassword(ctx, userID, "wrong_old_password", newPassword)
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "mật khẩu cũ không chính xác" {
			t.Errorf("unexpected error: %s", err.Error())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})

	t.Run("account not found", func(t *testing.T) {
		expectedGetQuery := regexp.QuoteMeta("SELECT id, email, username, COALESCE(password_hash,''), is_verified, COALESCE(is_admin,false), COALESCE(google_id,''), created_at, updated_at FROM accounts WHERE id = $1")

		mock.ExpectQuery(expectedGetQuery).WithArgs(userID).WillReturnError(sql.ErrNoRows)

		err := uc.ChangePassword(ctx, userID, oldPassword, newPassword)
		if err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != "tài khoản không tồn tại" {
			t.Errorf("unexpected error: %s", err.Error())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("sqlmock expectations not met: %s", err)
		}
	})
}
