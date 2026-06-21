package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	roomv1 "github.com/worktogether/services/playback-service/api/v1"
	deliveryGrpc "github.com/worktogether/services/playback-service/internal/delivery/grpc"
	delivery "github.com/worktogether/services/playback-service/internal/delivery/http"
	"github.com/worktogether/services/playback-service/internal/repository"
	"github.com/worktogether/services/playback-service/internal/usecase"
	"github.com/worktogether/services/playback-service/pkg/middleware"
)

func main() {
	log.Println("Bắt đầu khởi chạy playback-service...")

	// Đọc cấu hình từ biến môi trường
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "redis_password")

	jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
	port := getEnv("PORT", "8087")

	// Kết nối Redis với retry
	var rdb *redis.Client
	var err error
	for i := 0; i < 10; i++ {
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
			DB:       0,
		})
		
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = rdb.Ping(ctx).Err()
		cancel()
		
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối được với Redis (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Không thể kết nối đến Redis sau 10 lần thử: %v\n", err)
	}
	defer rdb.Close()

	log.Println("Kết nối cơ sở dữ liệu Redis thành công.")

	// Khởi tạo các lớp Layer
	repo := repository.NewRedisRepository(rdb)
	uc := usecase.NewPlaybackUsecase(repo)
	
	// Khởi tạo gRPC Client liên kết với room-service
	roomServiceAddr := getEnv("ROOM_SERVICE_GRPC", "room-service:50051")
	var conn *grpc.ClientConn
	for i := 0; i < 10; i++ {
		conn, err = grpc.Dial(roomServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			break
		}
		log.Printf("Chưa kết nối gRPC room-service (Thử lại %d/10): %v\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Không thể kết nối gRPC đến room-service: %v\n", err)
	}
	defer conn.Close()
	roomClient := roomv1.NewRoomInternalServiceClient(conn)
	log.Println("Khởi tạo kết nối gRPC sang room-service thành công.")
	
	// Khởi chạy WebSocket Hub
	hub := delivery.NewHub(uc, roomClient, jwtSecret)
	go hub.Run()

	// Khởi chạy gRPC Server nội bộ cho playback-service
	playbackGrpcPort := getEnv("GRPC_PORT", "50052")
	lisPlayback, err := net.Listen("tcp", ":"+playbackGrpcPort)
	if err != nil {
		log.Fatalf("Lỗi mở cổng gRPC playback: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	playbackGrpcServer := deliveryGrpc.NewPlaybackGrpcServer(uc, hub)
	roomv1.RegisterPlaybackInternalServiceServer(grpcServer, playbackGrpcServer)

	go func() {
		log.Printf("Playback gRPC Server đang lắng nghe tại cổng :%s...\n", playbackGrpcPort)
		if err := grpcServer.Serve(lisPlayback); err != nil {
			log.Fatalf("Lỗi khởi chạy gRPC server cho playback: %v\n", err)
		}
	}()

	// Khởi tạo Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Định nghĩa Routes
	// Nginx proxy: location ~ ^/api/v1/rooms/([^/]+)/playback/ws$
	r.GET("/api/v1/rooms/:room_id/playback/ws", delivery.ServePlaybackWS(hub))
	
	// REST API cho Playback State
	playbackGroup := r.Group("/api/v1/rooms/:room_id/playback")
	playbackGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		playbackGroup.GET("/state", func(c *gin.Context) {
			roomID := c.Param("room_id")
			state, err := uc.GetOrCreateState(c.Request.Context(), roomID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"data":    nil,
					"error": gin.H{
						"code":    "SERVER_ERROR",
						"message": err.Error(),
					},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    state,
				"error":   nil,
			})
		})

		playbackGroup.POST("/dj", func(c *gin.Context) {
			roomID := c.Param("room_id")
			userIDVal, exists := c.Get("userID")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
				return
			}
			userID := userIDVal.(string)

			// 1. Xác thực người gọi có phải là Host/Owner của phòng không
			res, err := roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
				RoomID: roomID,
				UserID: userID,
			})
			if err != nil || res == nil || !res.IsMember || res.Role != "OWNER" {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "FORBIDDEN",
						"message": "Chỉ Host của phòng mới có quyền bổ nhiệm Guest DJ.",
					},
				})
				return
			}

			var req struct {
				UserID          string `json:"user_id" binding:"required"`
				DurationSeconds int    `json:"duration_seconds" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
				return
			}

			// 2. Lưu trạng thái Guest DJ vào Redis
			ttl := time.Duration(req.DurationSeconds) * time.Second
			if err := uc.SetGuestDJ(c.Request.Context(), roomID, req.UserID, ttl); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}

			// 3. Broadcast sự kiện takeover qua Hub tới cả phòng
			endsAt := time.Now().Add(ttl).UnixNano() / int64(time.Millisecond)
			hub.BroadcastToRoom(roomID, "dj:takeover", gin.H{
				"user_id": req.UserID,
				"ends_at": endsAt,
			})

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"message": "Đã nhường quyền Guest DJ thành công.",
					"user_id": req.UserID,
					"ends_at": endsAt,
				},
			})
		})

		playbackGroup.DELETE("/dj", func(c *gin.Context) {
			roomID := c.Param("room_id")
			userIDVal, exists := c.Get("userID")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
				return
			}
			userID := userIDVal.(string)

			// 1. Xác thực người gọi có phải là Host/Owner của phòng không
			res, err := roomClient.VerifyRoomMember(c.Request.Context(), &roomv1.VerifyRoomMemberRequest{
				RoomID: roomID,
				UserID: userID,
			})
			if err != nil || res == nil || !res.IsMember || res.Role != "OWNER" {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "FORBIDDEN",
						"message": "Chỉ Host của phòng mới có quyền thu hồi Guest DJ.",
					},
				})
				return
			}

			// 2. Xóa trạng thái Guest DJ trong Redis
			if err := uc.ClearGuestDJ(c.Request.Context(), roomID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}

			// 3. Broadcast sự kiện released qua Hub tới cả phòng
			hub.BroadcastToRoom(roomID, "dj:released", gin.H{
				"reason": "revoked",
			})

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"message": "Đã thu hồi quyền Guest DJ thành công.",
				},
			})
		})
	}

	// Liveness & Readiness probe
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	log.Printf("Playback Service đang lắng nghe tại cổng :%s...\n", port)
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
