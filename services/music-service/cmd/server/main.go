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
	delivery "github.com/worktogether/services/music-service/internal/delivery/http"
	"github.com/worktogether/services/music-service/internal/repository"
	"github.com/worktogether/services/music-service/internal/usecase"
	"github.com/worktogether/services/music-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy music-service...")

	// Đọc cấu hình từ biến môi trường
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_music")

	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8085")

	// MinIO Configuration
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccess := getEnv("MINIO_ACCESS_KEY", "minio_admin")
	minioSecret := getEnv("MINIO_SECRET_KEY", "minio_password")
	minioBucket := getEnv("MINIO_BUCKET", "music")
	minioPublic := getEnv("MINIO_PUBLIC_URL", "http://localhost:9000") // URL public của MinIO cho client download

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
	uc := usecase.NewMusicUsecase(repo, minioEndpoint, minioAccess, minioSecret, minioBucket, minioPublic)
	handler := delivery.NewMusicHandler(uc)

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

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

	log.Printf("Music Service đang lắng nghe tại cổng :%s...\n", port)
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
