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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	roomv1 "github.com/worktogether/services/chat-service/api/v1"
	delivery "github.com/worktogether/services/chat-service/internal/delivery/http"
	deliveryRedis "github.com/worktogether/services/chat-service/internal/delivery/redis"
	"github.com/worktogether/services/chat-service/internal/repository"
	"github.com/worktogether/services/chat-service/internal/usecase"
	"github.com/worktogether/services/chat-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy chat-service...")

	// 1. Cấu hình Postgres
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_chat")

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
		log.Printf("Chưa kết nối được với PostgreSQL (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối đến PostgreSQL sau 10 lần thử: %v\n", err)
	}
	defer db.Close()
	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// 2. Cấu hình Redis
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	for i := 0; i < 10; i++ {
		err = rdb.Ping(context.Background()).Err()
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

	// 3. Khởi tạo gRPC Client liên kết với room-service
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

	// 4. Khởi tạo Layers và WebSockets Hub
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewChatUsecase(repo)
	hub := delivery.NewHub()
	hub.StartVibeTicker()
	handler := delivery.NewChatHandler(uc, hub, roomClient)

	// 5. Khởi chạy Redis Stream Worker (Ngắt socket khi bị kick/ban)
	worker := deliveryRedis.NewEventWorker(rdb, hub)
	go worker.Start(context.Background())

	// 6. Khởi chạy Gin HTTP Server
	port := getEnv("PORT", "8084")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// WebSocket & History endpoints bảo vệ bởi AuthMiddleware
	chatGroup := r.Group("/api/v1/rooms/:id/chat")
	chatGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		chatGroup.GET("/ws", handler.HandleWS)
		chatGroup.GET("/messages", handler.GetMessages)
		chatGroup.GET("/search", handler.SearchMessages)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Chat HTTP & WebSocket Server đang lắng nghe tại cổng :%s...\n", port)
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
