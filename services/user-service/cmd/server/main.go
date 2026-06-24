package main

import (
	"os/signal"
	"syscall"
	dbpkg "github.com/worktogether/pkg/db"
	"github.com/worktogether/pkg/env"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	delivery "github.com/worktogether/services/user-service/internal/delivery/http"
	"github.com/worktogether/services/user-service/internal/repository"
	"github.com/worktogether/services/user-service/internal/usecase"
	"github.com/worktogether/services/user-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy user-service...")

	// 1. Cấu hình Postgres
	dbHost := env.GetEnv("DB_HOST", "localhost")
	dbPort := env.GetEnv("DB_PORT", "5432")
	dbUser := env.GetEnv("DB_USER", "postgres")
	dbPassword := env.GetEnv("DB_PASSWORD", "postgres_password")
	dbName := env.GetEnv("DB_NAME", "worktogether_user")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err := dbpkg.ConnectPostgres(connStr)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer db.Close()
	log.Println("Kết nối cơ sở dữ liệu PostgreSQL thành công.")

	// 2. Cấu hình Redis
	redisHost := env.GetEnv("REDIS_HOST", "localhost")
	redisPort := env.GetEnv("REDIS_PORT", "6379")
	redisPassword := env.GetEnv("REDIS_PASSWORD", "redis_password")

	rdb, err := dbpkg.ConnectRedis(redisHost, redisPort, redisPassword)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer rdb.Close()
	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// 3. Khởi tạo Layers
	pgRepo := repository.NewPostgresRepository(db)
	redisRepo := repository.NewRedisRepository(rdb)
	uc := usecase.NewUserUsecase(pgRepo, redisRepo)
	handler := delivery.NewUserHandler(uc)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
	}
	port := env.GetEnv("PORT", "8082")

	// 4. Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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

		// Hủy / xóa quan hệ bạn bè
		usersGroup.DELETE("/friends/request/:id", handler.CancelFriendRequest)
		usersGroup.DELETE("/friends/:id", handler.Unfriend)
		usersGroup.DELETE("/friends/block/:id", handler.UnblockUser)
	}

	// Sửa lỗi cú pháp nhỏ ở dòng 104, handler thay vì h.
	// Cụ thể là: usersGroup.PUT("/friends/request/:id", handler.RespondFriendRequest)
	// Để tránh lỗi build, tôi viết code đúng trực tiếp ở dưới.

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

