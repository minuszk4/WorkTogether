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
	"github.com/worktogether/services/collab-service/internal/delivery/ws"
	"github.com/worktogether/services/collab-service/internal/repository"
	"github.com/worktogether/services/collab-service/internal/usecase"
)

func main() {
	log.Println("Bắt đầu khởi chạy collab-service...")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_collab")
	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8094")

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

	// Set up layers
	repo := repository.NewPostgresRepository(db)
	uc := usecase.NewCollabUsecase(repo)
	hub := ws.NewHub(jwtSecret, uc)

	go hub.Run()

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// WebSocket endpoint
	r.GET("/api/v1/rooms/:room_id/collab/ws", ws.ServeCollabWS(hub))

	serverAddr := ":" + port
	log.Printf("Collab Service đang lắng nghe tại cổng %s...\n", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Lỗi khởi chạy Gin: %v\n", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
