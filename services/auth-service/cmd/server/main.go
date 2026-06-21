package main

import (
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
)

func main() {
	log.Println("Bắt đầu khởi chạy auth-service...")

	// ── Đọc cấu hình ──────────────────────────────────────────────────
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_auth")

	jwtSecret := getEnv("JWT_SECRET", "worktogether_dev_secret")
	jwtExpMinsStr := getEnv("JWT_EXP_MINS", "15")
	jwtExpMins, err := strconv.Atoi(jwtExpMinsStr)
	if err != nil {
		jwtExpMins = 15
	}

	port := getEnv("PORT", "8081")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:4200")

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
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Auth Service đang lắng nghe tại cổng :%s...\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Lỗi khởi chạy HTTP server: %v\n", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
