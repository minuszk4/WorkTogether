package main

import (
	dbpkg "github.com/worktogether/pkg/db"
	"github.com/worktogether/pkg/env"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/worktogether/services/collab-service/internal/delivery/ws"
	"github.com/worktogether/services/collab-service/internal/repository"
	"github.com/worktogether/services/collab-service/internal/usecase"
)

func main() {
	log.Println("Bắt đầu khởi chạy collab-service...")

	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_collab")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8094")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err := dbpkg.ConnectPostgres(connStr)
	if err != nil {
		log.Fatalf("%v", err)
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

