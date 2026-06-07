package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/user-service/internal/domain"
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func (r *RedisRepository) SetPresence(ctx context.Context, userID string, p *domain.Presence) error {
	key := fmt.Sprintf("presence:user:%s", userID)
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// Đặt TTL là 90s, heartbeat định kỳ 30s
	// Nếu người dùng offline, chúng ta sẽ xóa key hoặc đặt status offline.
	// Cho phép key tự động hết hạn nếu không có heartbeat.
	return r.rdb.Set(ctx, key, data, 90*time.Second).Err()
}

func (r *RedisRepository) GetPresence(ctx context.Context, userID string) (*domain.Presence, error) {
	key := fmt.Sprintf("presence:user:%s", userID)
	data, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Không có key đồng nghĩa với Offline
			return &domain.Presence{
				Status:     "offline",
				CustomText: "",
				LastActive: time.Now().Unix(),
			}, nil
		}
		return nil, err
	}

	p := &domain.Presence{}
	if err := json.Unmarshal([]byte(data), p); err != nil {
		return nil, err
	}
	return p, nil
}
