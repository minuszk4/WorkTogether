package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dbpkg "github.com/worktogether/pkg/db"
	"github.com/worktogether/pkg/env"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	roomv1 "github.com/worktogether/services/music-service/api/v1"
	delivery "github.com/worktogether/services/music-service/internal/delivery/http"
	"github.com/worktogether/services/music-service/internal/repository"
	"github.com/worktogether/services/music-service/internal/usecase"
	"github.com/worktogether/services/music-service/pkg/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Bắt đầu khởi chạy music-service...")

	// Đọc cấu hình từ biến môi trường
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_music")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8085")

	// MinIO Configuration
	minioEndpoint := env.GetEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccess := env.GetEnv("MINIO_ACCESS_KEY", "minio_admin")
	minioSecret := env.GetEnv("MINIO_SECRET_KEY", "minio_password")
	minioBucket := env.GetEnv("MINIO_BUCKET", "music")
	minioPublic := env.GetEnv("MINIO_PUBLIC_URL", "http://localhost:9000") // URL public của MinIO cho client download

	// Kết nối DB với retry
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	var err error
	db, err := dbpkg.ConnectPostgres(connStr)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer db.Close()

	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// Khởi tạo các lớp Layer
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewMusicUsecase(repo, minioEndpoint, minioAccess, minioSecret, minioBucket, minioPublic)
	roomConn, err := grpc.Dial(env.GetEnv("ROOM_SERVICE_GRPC", "room-service:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Không thể kết nối gRPC đến room-service: %v", err)
	}
	defer roomConn.Close()
	handler := delivery.NewMusicHandler(uc, roomv1.NewRoomInternalServiceClient(roomConn))

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Định nghĩa Routes
	musicGroup := r.Group("/api/v1/music")
	musicGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		musicGroup.POST("/extract", handler.ExtractYoutube)
		musicGroup.POST("/upload", handler.UploadAudio)
		musicGroup.GET("/search", handler.Search)
		musicGroup.GET("/:id", handler.GetTrack)
		musicGroup.POST("/history", handler.LogPlayback)
		musicGroup.GET("/history/:room_id", handler.GetHistory)
		musicGroup.GET("/rooms/:room_id/stats", handler.GetStats)

		// Lyrics endpoints
		musicGroup.GET("/tracks/:track_id/lyrics", handler.GetLyrics)
		musicGroup.POST("/tracks/:track_id/lyrics", handler.SaveLyrics)

		// Bookmarks endpoints
		musicGroup.GET("/rooms/:room_id/bookmarks", handler.GetBookmarks)
		musicGroup.POST("/rooms/:room_id/bookmarks", handler.SaveBookmark)
		musicGroup.DELETE("/rooms/:room_id/bookmarks/:id", handler.DeleteBookmark)
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
