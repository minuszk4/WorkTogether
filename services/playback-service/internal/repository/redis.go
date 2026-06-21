package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/playback-service/internal/domain"
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func (r *RedisRepository) SavePlaybackState(ctx context.Context, roomID string, state *domain.PlaybackState) error {
	key := fmt.Sprintf("room:%s:playback", roomID)
	
	fields := map[string]interface{}{
		"state":            state.State,
		"current_track_id": state.CurrentTrackID,
		"position_ms":      strconv.Itoa(state.PositionMS),
		"updated_at":       strconv.FormatInt(state.UpdatedAt, 10),
		"title":            state.Title,
		"artist":           state.Artist,
		"thumbnail_url":    state.ThumbnailURL,
		"duration_ms":      strconv.Itoa(state.DurationMS),
		"source_url":       state.SourceURL,
	}

	err := r.rdb.HMSet(ctx, key, fields).Err()
	if err != nil {
		return err
	}

	// Đặt TTL cho trạng thái phát nhạc (24 giờ sau khi không hoạt động)
	r.rdb.Expire(ctx, key, 24*time.Hour)
	return nil
}

func (r *RedisRepository) GetPlaybackState(ctx context.Context, roomID string) (*domain.PlaybackState, error) {
	key := fmt.Sprintf("room:%s:playback", roomID)
	
	vals, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(vals) == 0 {
		return nil, nil // Chưa có trạng thái nào lưu
	}

	pos, _ := strconv.Atoi(vals["position_ms"])
	upTime, _ := strconv.ParseInt(vals["updated_at"], 10, 64)
	dur, _ := strconv.Atoi(vals["duration_ms"])

	return &domain.PlaybackState{
		State:          vals["state"],
		CurrentTrackID: vals["current_track_id"],
		PositionMS:     pos,
		UpdatedAt:      upTime,
		Title:          vals["title"],
		Artist:         vals["artist"],
		ThumbnailURL:   vals["thumbnail_url"],
		DurationMS:     dur,
		SourceURL:      vals["source_url"],
	}, nil
}

func (r *RedisRepository) SetGuestDJ(ctx context.Context, roomID string, userID string, ttl time.Duration) error {
	return r.rdb.Set(ctx, "room:"+roomID+":guest_dj", userID, ttl).Err()
}

func (r *RedisRepository) GetGuestDJ(ctx context.Context, roomID string) (string, error) {
	val, err := r.rdb.Get(ctx, "room:"+roomID+":guest_dj").Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (r *RedisRepository) ClearGuestDJ(ctx context.Context, roomID string) error {
	return r.rdb.Del(ctx, "room:"+roomID+":guest_dj").Err()
}

func (r *RedisRepository) SetPollActive(ctx context.Context, roomID string, active bool) error {
	return r.rdb.Set(ctx, "room:"+roomID+":poll_active", active, 24*time.Hour).Err()
}

func (r *RedisRepository) IsPollActive(ctx context.Context, roomID string) (bool, error) {
	val, err := r.rdb.Get(ctx, "room:"+roomID+":poll_active").Bool()
	if err == redis.Nil {
		return false, nil
	}
	return val, err
}

func (r *RedisRepository) VoteForTrack(ctx context.Context, roomID string, trackID string) error {
	return r.rdb.HIncrBy(ctx, "room:"+roomID+":poll_votes", trackID, 1).Err()
}

func (r *RedisRepository) GetPollVotes(ctx context.Context, roomID string) (map[string]int, error) {
	vals, err := r.rdb.HGetAll(ctx, "room:"+roomID+":poll_votes").Result()
	if err != nil {
		return nil, err
	}
	res := make(map[string]int)
	for k, v := range vals {
		num, _ := strconv.Atoi(v)
		res[k] = num
	}
	return res, nil
}

func (r *RedisRepository) ClearPoll(ctx context.Context, roomID string) error {
	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, "room:"+roomID+":poll_active")
	pipe.Del(ctx, "room:"+roomID+":poll_votes")
	_, err := pipe.Exec(ctx)
	return err
}
