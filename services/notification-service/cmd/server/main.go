package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	delivery "github.com/worktogether/services/notification-service/internal/delivery/http"
	"github.com/worktogether/services/notification-service/internal/repository"
	"github.com/worktogether/services/notification-service/internal/usecase"
	"github.com/worktogether/services/notification-service/internal/worker"
	"github.com/worktogether/services/notification-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy notification-service...")
	
	// Đọc cấu hình từ biến môi trường
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_notification")

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8089")

	// 1. Kết nối PostgreSQL với retry
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	var db *sql.DB
	var err error
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

	// 2. Kết nối Redis với retry
	var rdb *redis.Client
	for i := 0; i < 10; i++ {
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
			DB:       0,
		})
		
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = rdb.Ping(ctx).Err()
		cancel()
		
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối được với Redis (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Không thể kết nối đến Redis sau 10 lần thử: %v\n", err)
	}
	defer rdb.Close()
	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// Khởi tạo các lớp Layer
	pgRepo := repository.NewPostgresRepository(db)
	uc := usecase.NewNotificationUsecase(pgRepo)
	
	// Khởi chạy Redis Worker để lắng nghe notification trigger
	_ = worker.NewRedisWorker(rdb, uc, "stream:notification_trigger")

	handler := delivery.NewNotificationHandler(uc, jwtSecret)

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes
	// Nginx proxy: location /api/v1/notifications/stream
	r.GET("/api/v1/notifications/stream", handler.ServeSSE)
	
	// Các API REST cơ bản
	notiGroup := r.Group("/api/v1/notifications")
	notiGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		notiGroup.GET("/", handler.GetNotifications)
		notiGroup.PUT("/:id/read", handler.MarkRead)
		notiGroup.PUT("/read-all", handler.MarkAllRead)
		notiGroup.GET("/unread-count", handler.GetUnreadCount)
		
		// Endpoint thủ công để test
		notiGroup.POST("/trigger", handler.TriggerNotification)
	}

	// Liveness & Readiness probe
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Notification Service đang lắng nghe tại cổng :%s...\n", port)
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
