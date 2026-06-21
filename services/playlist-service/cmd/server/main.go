package main

import (
	"database/sql"
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
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_playlist")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := getEnv("PORT", "8086")

	// Kết nối DB với retry
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
		log.Printf("Chưa kết nối được với Database (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Không thể kết nối đến Database sau 10 lần thử: %v\n", err)
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

	log.Printf("Playlist Service đang lắng nghe tại cổng :%s...\n", port)
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
