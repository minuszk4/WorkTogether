package main

import (
	"context"
	"os/signal"
	"syscall"
	dbpkg "github.com/worktogether/pkg/db"
	"github.com/worktogether/pkg/env"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	delivery "github.com/worktogether/services/playlist-service/internal/delivery/http"
	"github.com/worktogether/services/playlist-service/internal/repository"
	"github.com/worktogether/services/playlist-service/internal/usecase"
	"github.com/worktogether/services/playlist-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy playlist-service...")

	// Đọc cấu hình từ biến môi trường
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_playlist")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8086")

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
	uc := usecase.NewPlaylistUsecase(repo)
	handler := delivery.NewPlaylistHandler(uc)

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Định nghĩa Routes
	playlistGroup := r.Group("/api/v1/playlists")
	playlistGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		playlistGroup.POST("/", handler.CreatePlaylist)
		playlistGroup.GET("/room/:room_id", handler.GetRoomPlaylists)
		playlistGroup.GET("/user", handler.GetUserPlaylists)
		playlistGroup.DELETE("/:id", handler.DeletePlaylist)
		
		// Tracks in playlist
		playlistGroup.POST("/:id/tracks", handler.AddTrack)
		playlistGroup.GET("/:id/tracks", handler.GetTracks)
		playlistGroup.DELETE("/:id/tracks/:item_id", handler.RemoveTrack)
		playlistGroup.PUT("/:id/tracks/:item_id/move", handler.MoveTrack)
		playlistGroup.POST("/tracks/:item_id/vote", handler.VoteTrack)
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

