package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	delivery "github.com/worktogether/services/playback-service/internal/delivery/http"
	"github.com/worktogether/services/playback-service/internal/repository"
	"github.com/worktogether/services/playback-service/internal/usecase"
	"github.com/worktogether/services/playback-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy playback-service...")

	// Đọc cấu hình từ biến môi trường
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8087")

	// Kết nối Redis với retry
	var rdb *redis.Client
	var err error
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
	repo := repository.NewRedisRepository(rdb)
	uc := usecase.NewPlaybackUsecase(repo)
	
	// Khởi chạy WebSocket Hub
	hub := delivery.NewHub(uc, jwtSecret)
	go hub.Run()

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes
	// Nginx proxy: location ~ ^/api/v1/rooms/([^/]+)/playback/ws$
	r.GET("/api/v1/rooms/:room_id/playback/ws", delivery.ServePlaybackWS(hub))
	
	// REST API cho Playback State
	playbackGroup := r.Group("/api/v1/rooms/:room_id/playback")
	playbackGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		playbackGroup.GET("/state", func(c *gin.Context) {
			roomID := c.Param("room_id")
			state, err := uc.GetOrCreateState(c.Request.Context(), roomID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"data":    nil,
					"error": gin.H{
						"code":    "SERVER_ERROR",
						"message": err.Error(),
					},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    state,
				"error":   nil,
			})
		})
	}

	// Liveness & Readiness probe
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Playback Service đang lắng nghe tại cổng :%s...\n", port)
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
