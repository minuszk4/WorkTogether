package main

import (
	"os/signal"
	"syscall"
	"github.com/worktogether/pkg/env"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	roomv1 "github.com/worktogether/services/chat-service/api/v1"
	delivery "github.com/worktogether/services/chat-service/internal/delivery/http"
	deliveryRedis "github.com/worktogether/services/chat-service/internal/delivery/redis"
	"github.com/worktogether/services/chat-service/internal/repository"
	"github.com/worktogether/services/chat-service/internal/tracing"
	"github.com/worktogether/services/chat-service/internal/usecase"
	"github.com/worktogether/services/chat-service/pkg/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	log.Println("Bắt đầu khởi chạy chat-service...")

	// 0. Khởi tạo Tracing
	shutdown, err := tracing.InitTracer("chat-service")
	if err != nil {
		log.Printf("Lỗi khởi tạo tracer: %v\n", err)
	} else {
		defer shutdown(context.Background())
	}

	// 1. Cấu hình Postgres
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_chat")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	var db *sql.DB
	var sqlErr error
	for i := 0; i < 10; i++ {
		db, sqlErr = sql.Open("pgx", connStr)
		if sqlErr == nil {
			sqlErr = db.Ping()
			if sqlErr == nil {
				break
			}
		}
		log.Printf("Chưa kết nối được với PostgreSQL (Thử lại %d/10): %v\n", i+1, sqlErr)
		time.Sleep(3 * time.Second)
	}
	if sqlErr != nil {
		log.Fatalf("Không thể kết nối đến PostgreSQL sau 10 lần thử: %v\n", sqlErr)
	}
	defer db.Close()
	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// 2. Cấu hình Redis
	redisHost := env.GetEnv("REDIS_HOST", "localhost")
	redisPort := env.GetEnv("REDIS_PORT", "6379")
	redisPassword := env.GetEnv("REDIS_PASSWORD", "redis_password")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	var redisErr error
	for i := 0; i < 10; i++ {
		redisErr = rdb.Ping(context.Background()).Err()
		if redisErr == nil {
			break
		}
		log.Printf("Chưa kết nối được với Redis (Thử lại %d/10): %v\n", i+1, redisErr)
		time.Sleep(3 * time.Second)
	}
	if redisErr != nil {
		log.Fatalf("Không thể kết nối đến Redis sau 10 lần thử: %v\n", redisErr)
	}
	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// 3. Khởi tạo gRPC Client liên kết với room-service
	roomServiceAddr := env.GetEnv("ROOM_SERVICE_GRPC", env.GetEnv("ROOM_SERVICE_GRPC_ADDR", "localhost:50051"))
	var conn *grpc.ClientConn
	var grpcErr error
	for i := 0; i < 10; i++ {
		conn, grpcErr = grpc.Dial(roomServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if grpcErr == nil {
			break
		}
		log.Printf("Chưa kết nối gRPC room-service (Thử lại %d/10): %v\n", i+1, grpcErr)
		time.Sleep(3 * time.Second)
	}
	if grpcErr != nil {
		log.Fatalf("Không thể kết nối gRPC đến room-service: %v\n", grpcErr)
	}
	defer conn.Close()
	roomClient := roomv1.NewRoomInternalServiceClient(conn)
	log.Println("Khởi tạo kết nối gRPC sang room-service thành công.")

	// 4. Khởi tạo Layers và WebSockets Hub
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewChatUsecase(repo, rdb)
	hub := delivery.NewHub(rdb)
	hub.StartVibeTicker()
	handler := delivery.NewChatHandler(uc, hub, roomClient)

	// 5. Khởi chạy Redis Stream Worker (Ngắt socket khi bị kick/ban)
	worker := deliveryRedis.NewEventWorker(rdb, hub)
	go worker.Start(context.Background())

	// 6. Khởi chạy Gin HTTP Server
	port := env.GetEnv("PORT", "8084")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(otelgin.Middleware("chat-service"))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// WebSocket endpoint KHÔNG bảo vệ bởi AuthMiddleware (Xác thực token trong HandleWS)
	r.GET("/api/v1/rooms/:id/chat/ws", handler.HandleWS)

	// History endpoints bảo vệ bởi AuthMiddleware
	chatGroup := r.Group("/api/v1/rooms/:id/chat")
	chatGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		chatGroup.GET("/messages", handler.GetMessages)
		chatGroup.GET("/search", handler.SearchMessages)
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

