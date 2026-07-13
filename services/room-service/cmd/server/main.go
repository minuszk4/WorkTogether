package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/worktogether/pkg/env"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	roomv1 "github.com/worktogether/services/room-service/api/v1"
	deliveryGrpc "github.com/worktogether/services/room-service/internal/delivery/grpc"
	deliveryHttp "github.com/worktogether/services/room-service/internal/delivery/http"
	"github.com/worktogether/services/room-service/internal/repository"
	"github.com/worktogether/services/room-service/internal/tracing"
	"github.com/worktogether/services/room-service/internal/usecase"
	"github.com/worktogether/services/room-service/pkg/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Bắt đầu khởi chạy room-service...")

	// 0. Khởi tạo Tracing
	shutdown, err := tracing.InitTracer("room-service")
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
	dbName := env.GetEnv("DB_NAME", "worktogether_room")

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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// 1.5 Khởi tạo Redis
	redisHost := env.GetEnv("REDIS_HOST", "localhost")
	redisPort := env.GetEnv("REDIS_PORT", "6379")
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	// 2. Khởi tạo Layers
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewRoomUsecase(repo, rdb)

	// 3. Khởi chạy gRPC Server nội bộ
	grpcPort := env.GetEnv("GRPC_PORT", "50051")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Lỗi mở cổng gRPC: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()
	roomGrpcServer := deliveryGrpc.NewRoomGrpcServer(uc)
	roomv1.RegisterRoomInternalServiceServer(grpcServer, roomGrpcServer)

	go func() {
		log.Printf("Room gRPC Server đang lắng nghe tại cổng :%s...\n", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Lỗi khởi chạy gRPC server: %v\n", err)
		}
	}()

	// 4. Khởi chạy HTTP Web Server
	port := env.GetEnv("PORT", "8083")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	handler := deliveryHttp.NewRoomHandler(uc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(otelgin.Middleware("room-service"))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Định nghĩa Routes bảo vệ bằng AuthMiddleware
	roomsGroup := r.Group("/api/v1/rooms")
	roomsGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		roomsGroup.POST("", handler.CreateRoom)
		roomsGroup.DELETE("/:id", handler.DeleteRoom)
		roomsGroup.GET("", handler.GetRooms)
		roomsGroup.GET("/invite/:code", handler.GetRoomByInviteCode)
		roomsGroup.GET("/:id", handler.GetRoomByID)
		roomsGroup.PUT("/:id/settings", handler.UpdateRoomSettings)
		roomsGroup.PUT("/:id/mode", handler.UpdateRoomMode)
		roomsGroup.GET("/session-templates", handler.ListSessionTemplates)
		roomsGroup.GET("/me/actions", handler.GetPersonalActionItems)
		roomsGroup.GET("/me/sessions", handler.GetMySessionRecaps)
		roomsGroup.GET("/:id/session", handler.GetActiveSession)
		roomsGroup.GET("/:id/events", handler.ListRoomEvents)
		roomsGroup.POST("/:id/events", handler.CreateRoomEvent)
		roomsGroup.POST("/:id/sessions", handler.StartSession)
		roomsGroup.GET("/:id/sessions/:session_id", handler.GetSessionWorkspace)
		roomsGroup.PUT("/:id/sessions/:session_id/complete", handler.CompleteSession)
		roomsGroup.POST("/:id/sessions/:session_id/agenda", handler.AddAgendaItem)
		roomsGroup.PUT("/:id/sessions/:session_id/agenda/:item_id", handler.UpdateAgendaItem)
		roomsGroup.POST("/:id/sessions/:session_id/actions", handler.AddActionItem)
		roomsGroup.PUT("/:id/sessions/:session_id/actions/:action_id", handler.UpdateActionItem)
		roomsGroup.GET("/:id/members", handler.GetMembers)
		roomsGroup.POST("/:id/join", handler.JoinRoom)
		roomsGroup.POST("/:id/leave", handler.LeaveRoom)

		// Phân quyền
		roomsGroup.POST("/:id/roles", handler.CreateRole)
		roomsGroup.PUT("/:id/members/:user_id/role", handler.AssignRole)

		// Vi phạm
		roomsGroup.POST("/:id/members/:user_id/kick", handler.KickMember)
		roomsGroup.POST("/:id/members/:user_id/ban", handler.BanMember)
		roomsGroup.POST("/:id/members/:user_id/mute", handler.MuteMember)
		roomsGroup.POST("/:id/members/:user_id/unmute", handler.UnmuteMember)

		// Subrooms
		roomsGroup.POST("/:id/subrooms", handler.CreateSubRoom)
		roomsGroup.GET("/:id/subrooms", handler.GetSubRooms)
		roomsGroup.PUT("/:id/members/:user_id/move", handler.MoveMember)
	}

	adminGroup := r.Group("/api/v1/admin")
	adminGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		adminGroup.GET("/rooms", handler.AdminListRooms)
		adminGroup.GET("/audit", handler.AdminListAuditEvents)
		adminGroup.GET("/rooms/:id/members", handler.AdminListMembers)
		adminGroup.PUT("/rooms/:id", handler.AdminUpdateRoom)
		adminGroup.DELETE("/rooms/:id", handler.AdminDeleteRoom)
		adminGroup.DELETE("/rooms/:id/members/:user_id", handler.AdminRemoveMember)
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
