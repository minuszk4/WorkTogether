package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	delivery "github.com/worktogether/services/user-service/internal/delivery/http"
	"github.com/worktogether/services/user-service/internal/repository"
	"github.com/worktogether/services/user-service/internal/usecase"
	"github.com/worktogether/services/user-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy user-service...")

	// 1. Cấu hình Postgres
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres_password")
	dbName := getEnv("DB_NAME", "worktogether_user")

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

	// 2. Cấu hình Redis
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	for i := 0; i < 10; i++ {
		err = rdb.Ping(context.Background()).Err()
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối được với Redis (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối đến Redis sau 10 lần thử: %v\n", err)
	}
	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// 3. Khởi tạo Layers
	pgRepo := repository.NewPostgresRepository(db)
	redisRepo := repository.NewRedisRepository(rdb)
	uc := usecase.NewUserUsecase(pgRepo, redisRepo)
	handler := delivery.NewUserHandler(uc)

	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8082")

	// 4. Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes bảo vệ bằng AuthMiddleware
	usersGroup := r.Group("/api/v1/users")
	usersGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		usersGroup.GET("/profile", handler.GetProfile)       // GET /users/profile (own)
		usersGroup.GET("/profile/:id", handler.GetProfile)   // GET /users/profile/:id
		usersGroup.GET("/:id/profile", handler.GetProfile)   // GET /users/:id/profile (REST style)
		usersGroup.PUT("/profile", handler.UpdateProfile)
		usersGroup.PUT("/status", handler.UpdatePresence)

		// Bạn bè
		usersGroup.GET("/friends", handler.GetFriends)
		usersGroup.POST("/friends/request", handler.SendFriendRequest)
		usersGroup.PUT("/friends/request/:id", handler.RespondFriendRequest)
		usersGroup.POST("/friends/block", handler.BlockUser)
	}

	// Sửa lỗi cú pháp nhỏ ở dòng 104, handler thay vì h.
	// Cụ thể là: usersGroup.PUT("/friends/request/:id", handler.RespondFriendRequest)
	// Để tránh lỗi build, tôi viết code đúng trực tiếp ở dưới.

	// Liveness & Readiness probe
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("User Service đang lắng nghe tại cổng :%s...\n", port)
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
