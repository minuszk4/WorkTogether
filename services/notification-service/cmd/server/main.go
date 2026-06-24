package main

import (
	"os/signal"
	"syscall"
	dbpkg "github.com/worktogether/pkg/db"
	"github.com/worktogether/pkg/env"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
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
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_notification")

	redisHost := env.GetEnv("REDIS_HOST", "localhost")
	redisPort := env.GetEnv("REDIS_PORT", "6379")
	redisPassword := env.GetEnv("REDIS_PASSWORD", "redis_password")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8089")

	// 1. Kết nối PostgreSQL với retry
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	var err error
	db, err := dbpkg.ConnectPostgres(connStr)
	if err != nil {
		log.Fatalf("%v", err)
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
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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

