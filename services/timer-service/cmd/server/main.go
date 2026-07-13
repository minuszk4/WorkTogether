package main

import (
	"context"
	"fmt"
	"github.com/worktogether/pkg/env"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	v1 "github.com/worktogether/services/timer-service/api/v1"
	"github.com/worktogether/services/timer-service/internal/delivery/ws"
	"github.com/worktogether/services/timer-service/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Bắt đầu khởi chạy timer-service...")

	redisHost := env.GetEnv("REDIS_HOST", "localhost")
	redisPort := env.GetEnv("REDIS_PORT", "6379")
	redisPassword := env.GetEnv("REDIS_PASSWORD", "redis_password")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8093")
	playbackServiceGrpc := env.GetEnv("PLAYBACK_SERVICE_GRPC", "playback-service:50052")
	roomServiceGrpc := env.GetEnv("ROOM_SERVICE_GRPC", env.GetEnv("ROOM_SERVICE_GRPC_ADDR", "room-service:50051"))

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
		conn, err = grpc.Dial(playbackServiceGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
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

	var roomConn *grpc.ClientConn
	for i := 0; i < 10; i++ {
		roomConn, err = grpc.Dial(roomServiceGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối gRPC room-service (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối gRPC đến room-service: %v\n", err)
	}
	defer roomConn.Close()
	roomClient := v1.NewRoomInternalServiceClient(roomConn)
	log.Println("Khởi tạo kết nối gRPC sang room-service thành công.")

	// Set up Hub and Usecase
	hub := ws.NewHub(jwtSecret, roomClient)
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
