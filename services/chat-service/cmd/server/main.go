package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/worktogether/pkg/env"
	"github.com/worktogether/pkg/translation"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	roomv1 "github.com/worktogether/services/chat-service/api/v1"
	delivery "github.com/worktogether/services/chat-service/internal/delivery/http"
	deliveryRedis "github.com/worktogether/services/chat-service/internal/delivery/redis"
	"github.com/worktogether/services/chat-service/internal/repository"
	"github.com/worktogether/services/chat-service/internal/tracing"
	"github.com/worktogether/services/chat-service/internal/usecase"
	"github.com/worktogether/services/chat-service/pkg/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	var speechTranscriber translation.Transcriber
	hub.StartRedisSubscriber()
	if credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credentialsPath != "" {
		translator, err := translation.NewGoogleTranslator(credentialsPath, nil)
		if err != nil {
			log.Printf("Realtime translation disabled: %v", err)
		} else {
			hub.EnableTranslation(context.Background(), envInt("TRANSLATION_QUEUE_SIZE", 64), time.Duration(envInt("TRANSLATION_TIMEOUT_MS", 1200))*time.Millisecond, envInt("TRANSLATION_MAX_CHARS_PER_MINUTE", 30000), translator.Translate)
			log.Println("Realtime translation worker enabled.")
		}
		transcriber, err := translation.NewGoogleSpeechTranscriber(credentialsPath, nil)
		if err != nil {
			log.Printf("Voice captions disabled: %v", err)
		} else {
			speechTranscriber = transcriber.Transcribe
			log.Println("Google voice captions enabled.")
		}
	}
	hub.StartVibeTicker()
	handler := delivery.NewChatHandler(uc, hub, roomClient)
	if speechTranscriber != nil {
		handler.EnableCaptionTranscription(speechTranscriber)
	}

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
		chatGroup.POST("/subtitles", handler.TranscribeCaption)
		chatGroup.DELETE("/messages/:message_id", handler.AdminDeleteMessage)
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

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
