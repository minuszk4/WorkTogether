package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	roomv1 "github.com/worktogether/services/room-service/api/v1"
	deliveryGrpc "github.com/worktogether/services/room-service/internal/delivery/grpc"
	deliveryHttp "github.com/worktogether/services/room-service/internal/delivery/http"
	"github.com/worktogether/services/room-service/internal/repository"
	"github.com/worktogether/services/room-service/internal/usecase"
	"github.com/worktogether/services/room-service/pkg/middleware"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Bắt đầu khởi chạy room-service...")

	// 1. Cấu hình Postgres
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_room")

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

	// 2. Khởi tạo Layers
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewRoomUsecase(repo)

	// 3. Khởi chạy gRPC Server nội bộ
	grpcPort := getEnv("GRPC_PORT", "50051")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Lỗi mở cổng gRPC: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	roomGrpcServer := deliveryGrpc.NewRoomGrpcServer(uc)
	roomv1.RegisterRoomInternalServiceServer(grpcServer, roomGrpcServer)

	go func() {
		log.Printf("Room gRPC Server đang lắng nghe tại cổng :%s...\n", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Lỗi khởi chạy gRPC server: %v\n", err)
		}
	}()

	// 4. Khởi chạy HTTP Web Server
	port := getEnv("PORT", "8083")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	handler := deliveryHttp.NewRoomHandler(uc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes bảo vệ bằng AuthMiddleware
	roomsGroup := r.Group("/api/v1/rooms")
	roomsGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		roomsGroup.POST("", handler.CreateRoom)
		roomsGroup.GET("", handler.GetRooms)
		roomsGroup.GET("/invite/:code", handler.GetRoomByInviteCode)
		roomsGroup.GET("/:id", handler.GetRoomByID)
		roomsGroup.PUT("/:id/settings", handler.UpdateRoomSettings)
		roomsGroup.GET("/:id/members", handler.GetMembers)
		roomsGroup.POST("/:id/join", handler.JoinRoom)
		roomsGroup.POST("/:id/leave", handler.LeaveRoom)

		// Phân quyền
		roomsGroup.POST("/:id/roles", handler.CreateRole)
		roomsGroup.PUT("/:id/members/:user_id/role", handler.AssignRole)

		// Vi phạm
		roomsGroup.POST("/:id/members/:user_id/kick", handler.KickMember)
		roomsGroup.POST("/:id/members/:user_id/ban", handler.BanMember)

		// Subrooms
		roomsGroup.POST("/:id/subrooms", handler.CreateSubRoom)
		roomsGroup.GET("/:id/subrooms", handler.GetSubRooms)
		roomsGroup.PUT("/:id/members/:user_id/move", handler.MoveMember)
	}

	// Liveness & Readiness probe
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Room HTTP Server đang lắng nghe tại cổng :%s...\n", port)
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
