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
	roomv1 "github.com/worktogether/services/voice-service/api/v1"
	delivery "github.com/worktogether/services/voice-service/internal/delivery/http"
	"github.com/worktogether/services/voice-service/internal/usecase"
	"github.com/worktogether/services/voice-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy voice-service...")

	// 1. Cấu hình Redis
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	var err error
	for i := 0; i < 10; i++ {
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
	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// 2. Khởi tạo gRPC Client liên kết với room-service
	roomServiceAddr := getEnv("ROOM_SERVICE_GRPC", getEnv("ROOM_SERVICE_GRPC_ADDR", "localhost:50051"))
	var conn *grpc.ClientConn
	for i := 0; i < 10; i++ {
		conn, err = grpc.Dial(roomServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối gRPC room-service (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối gRPC đến room-service: %v\n", err)
	}
	defer conn.Close()
	roomClient := roomv1.NewRoomInternalServiceClient(conn)
	log.Println("Khởi tạo kết nối gRPC sang room-service thành công.")

	// 3. Cấu hình LiveKit
	livekitURL := getEnv("LIVEKIT_URL", "ws://localhost:7880")
	livekitKey := getEnv("LIVEKIT_API_KEY", "devkey")
	livekitSecret := getEnv("LIVEKIT_API_SECRET", "worktogether_livekit_dev_secret_1234567890")

	uc := usecase.NewVoiceUsecase(livekitURL, livekitKey, livekitSecret, rdb)
	handler := delivery.NewVoiceHandler(uc, roomClient, livekitURL)

	port := getEnv("PORT", "8088")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}

	// 4. Khởi chạy Gin HTTP Server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes
	voiceGroup := r.Group("/api/v1/voice")
	{
		// Lấy token thoại yêu cầu đăng nhập
		voiceGroup.GET("/rooms/:room_id/token", middleware.AuthMiddleware(jwtSecret), handler.GetToken)
		
		// Webhook từ LiveKit (không dùng AuthMiddleware vì được gọi bởi LiveKit Server)
		voiceGroup.POST("/webhooks", handler.HandleWebhook)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Voice HTTP Server đang lắng nghe tại cổng :%s...\n", port)
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
