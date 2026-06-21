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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "github.com/worktogether/services/timer-service/api/v1"
	"github.com/worktogether/services/timer-service/internal/delivery/ws"
	"github.com/worktogether/services/timer-service/internal/usecase"
)

func main() {
	log.Println("Bắt đầu khởi chạy timer-service...")

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")
	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8093")
	playbackServiceGrpc := getEnv("PLAYBACK_SERVICE_GRPC", "playback-service:50052")

	// Connect to Redis with retry
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

	// Connect to playback gRPC server
	var conn *grpc.ClientConn
	for i := 0; i < 10; i++ {
		conn, err = grpc.Dial(playbackServiceGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối gRPC playback-service (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối gRPC đến playback-service: %v\n", err)
	}
	defer conn.Close()
	playbackClient := v1.NewPlaybackInternalServiceClient(conn)
	log.Println("Khởi tạo kết nối gRPC sang playback-service thành công.")

	// Set up Hub and Usecase
	hub := ws.NewHub(jwtSecret)
	uc := usecase.NewTimerUsecase(rdb, playbackClient, hub)
	hub.Usecase = uc

	go hub.Run()

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// WebSocket endpoint
	r.GET("/api/v1/rooms/:room_id/timer/ws", ws.ServeTimerWS(hub))

	serverAddr := ":" + port
	log.Printf("Timer Service đang lắng nghe tại cổng %s...\n", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Lỗi khởi chạy Gin: %v\n", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
