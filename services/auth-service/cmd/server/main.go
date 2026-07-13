package main

import (
	"context"
	"os/signal"
	"syscall"
	"github.com/worktogether/pkg/env"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	delivery "github.com/worktogether/services/auth-service/internal/delivery/http"
	"github.com/worktogether/services/auth-service/internal/repository"
	"github.com/worktogether/services/auth-service/internal/usecase"
	"github.com/worktogether/services/auth-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy auth-service...")

	// ── Đọc cấu hình ──────────────────────────────────────────────────
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_auth")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	jwtExpMinsStr := env.GetEnv("JWT_EXP_MINS", "15")
	jwtExpMins, err := strconv.Atoi(jwtExpMinsStr)
	if err != nil {
		jwtExpMins = 15
	}

	port := env.GetEnv("PORT", "8081")
	frontendURL := env.GetEnv("FRONTEND_URL", "http://localhost:4200")

	// ── Kết nối PostgreSQL ────────────────────────────────────────────
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	var db *sql.DB
	for i := 0; i < 10; i++ {
		db, err = sql.Open("pgx", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Chưa kết nối được với Database (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối đến Database sau 10 lần thử: %v\n", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// ── Khởi tạo các Layer ────────────────────────────────────────────
	repo := repository.NewPostgresRepository(db)
	emailSvc := usecase.NewEmailService()
	uc := usecase.NewAuthUsecase(repo, emailSvc, jwtSecret, jwtExpMins)
	googleCfg := usecase.NewGoogleOAuthConfig()
	handler := delivery.NewAuthHandler(uc, googleCfg, frontendURL)

	// Log trạng thái tích hợp
	if emailSvc.IsConfigured() {
		log.Println("✅ Email SMTP đã được cấu hình")
	} else {
		log.Println("⚠️  Email SMTP chưa cấu hình — dùng console log thay thế")
	}
	if usecase.IsGoogleConfigured() {
		log.Println("✅ Google OAuth đã được cấu hình")
	} else {
		log.Println("⚠️  Google OAuth chưa cấu hình — endpoint /google sẽ trả 503")
	}

	// ── Gin Router ────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/register", handler.Register)
		authGroup.GET("/verify-email", handler.VerifyEmail)
		authGroup.POST("/verify-email", handler.VerifyEmail)
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.Refresh)
		authGroup.POST("/logout", handler.Logout)

		// Google OAuth
		authGroup.GET("/google", handler.GoogleLogin)
		authGroup.GET("/google/callback", handler.GoogleCallback)

		// Password Management
		authGroup.POST("/forgot-password", handler.ForgotPassword)
		authGroup.POST("/reset-password", handler.ResetPassword)
		authGroup.POST("/change-password", middleware.AuthMiddleware(jwtSecret), handler.ChangePassword)
		authGroup.GET("/admin/accounts", middleware.AuthMiddleware(jwtSecret), handler.AdminListAccounts)
		authGroup.PUT("/admin/accounts/:id", middleware.AuthMiddleware(jwtSecret), handler.AdminSetAccountRole)
		authGroup.PUT("/admin/accounts/:id/suspension", middleware.AuthMiddleware(jwtSecret), handler.AdminSetAccountSuspension)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("HTTP Server đang lắng nghe tại cổng :%s...\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Lỗi khởi chạy HTTP server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Đang tắt server (Graceful Shutdown)...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server đã thoát an toàn.")
}
