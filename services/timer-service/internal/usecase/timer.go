package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	v1 "github.com/worktogether/services/timer-service/api/v1"
)

type PomodoroState struct {
	Status           string `json:"status"` // "focus", "break", "paused", "idle"
	PausedStatus     string `json:"paused_status,omitempty"`
	DurationSeconds  int    `json:"duration_seconds"`
	BreakSeconds     int    `json:"break_seconds"`
	RemainingSeconds int    `json:"remaining_seconds"`
	EndsAt           int64  `json:"ends_at"` // Epoch milliseconds
	CurrentCycle     int    `json:"current_cycle"`
	TotalCycles      int    `json:"total_cycles"`
}

type Broadcaster interface {
	BroadcastToRoom(roomID string, event string, data interface{})
}

type TimerUsecase struct {
	redisClient    *redis.Client
	playbackClient v1.PlaybackInternalServiceClient
	broadcaster    Broadcaster
	tickers        map[string]chan struct{}
	tickersMu      sync.Mutex
}

func NewTimerUsecase(redisClient *redis.Client, playbackClient v1.PlaybackInternalServiceClient, broadcaster Broadcaster) *TimerUsecase {
	return &TimerUsecase{
		redisClient:    redisClient,
		playbackClient: playbackClient,
		broadcaster:    broadcaster,
		tickers:        make(map[string]chan struct{}),
	}
}

func (uc *TimerUsecase) getRedisKey(roomID string) string {
	return fmt.Sprintf("room:%s:pomodoro", roomID)
}

func (uc *TimerUsecase) GetState(ctx context.Context, roomID string) (*PomodoroState, error) {
	key := uc.getRedisKey(roomID)
	data, err := uc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Default idle state
		return &PomodoroState{
			Status: "idle",
		}, nil
	} else if err != nil {
		return nil, err
	}

	var state PomodoroState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (uc *TimerUsecase) saveState(ctx context.Context, roomID string, state *PomodoroState) error {
	key := uc.getRedisKey(roomID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return uc.redisClient.Set(ctx, key, data, 0).Err()
}

func (uc *TimerUsecase) StartTimer(ctx context.Context, roomID string, duration int, breakSec int, cycles int) error {
	if duration <= 0 {
		duration = 1500 // 25 mins
	}
	if breakSec <= 0 {
		breakSec = 300 // 5 mins
	}
	if cycles <= 0 {
		cycles = 4
	}

	state := &PomodoroState{
		Status:           "focus",
		DurationSeconds:  duration,
		BreakSeconds:     breakSec,
		RemainingSeconds: duration,
		EndsAt:           time.Now().Add(time.Duration(duration) * time.Second).UnixNano() / int64(time.Millisecond),
		CurrentCycle:     1,
		TotalCycles:      cycles,
	}

	if err := uc.saveState(ctx, roomID, state); err != nil {
		return err
	}

	uc.startTicker(roomID)
	// Resume music when focus starts
	go uc.callPlaybackService(roomID, "resume")

	uc.broadcaster.BroadcastToRoom(roomID, "timer:sync", state)
	return nil
}

func (uc *TimerUsecase) PauseTimer(ctx context.Context, roomID string) error {
	state, err := uc.GetState(ctx, roomID)
	if err != nil {
		return err
	}

	if state.Status == "focus" || state.Status == "break" {
		state.PausedStatus = state.Status
		state.Status = "paused"
		state.EndsAt = 0
		if err := uc.saveState(ctx, roomID, state); err != nil {
			return err
		}

		uc.stopTicker(roomID)
		// Pause music on pause timer
		go uc.callPlaybackService(roomID, "pause")

		uc.broadcaster.BroadcastToRoom(roomID, "timer:sync", state)
	}

	return nil
}

func (uc *TimerUsecase) ResumeTimer(ctx context.Context, roomID string) error {
	state, err := uc.GetState(ctx, roomID)
	if err != nil {
		return err
	}

	if state.Status == "paused" {
		if state.PausedStatus != "" {
			state.Status = state.PausedStatus
		} else {
			state.Status = "focus"
		}
		state.EndsAt = time.Now().Add(time.Duration(state.RemainingSeconds) * time.Second).UnixNano() / int64(time.Millisecond)
		if err := uc.saveState(ctx, roomID, state); err != nil {
			return err
		}

		uc.startTicker(roomID)
		if state.Status == "focus" {
			go uc.callPlaybackService(roomID, "resume")
		} else if state.Status == "break" {
			go uc.callPlaybackService(roomID, "pause")
		}

		uc.broadcaster.BroadcastToRoom(roomID, "timer:sync", state)
	}

	return nil
}

func (uc *TimerUsecase) StopTimer(ctx context.Context, roomID string) error {
	state := &PomodoroState{
		Status: "idle",
	}

	if err := uc.saveState(ctx, roomID, state); err != nil {
		return err
	}

	uc.stopTicker(roomID)
	// Pause music on stop
	go uc.callPlaybackService(roomID, "pause")

	uc.broadcaster.BroadcastToRoom(roomID, "timer:sync", state)
	return nil
}

func (uc *TimerUsecase) RegisterTickerIfNeeded(ctx context.Context, roomID string) {
	state, err := uc.GetState(ctx, roomID)
	if err != nil || state == nil {
		return
	}

	if state.Status == "focus" || state.Status == "break" {
		uc.startTicker(roomID)
	}
}

func (uc *TimerUsecase) startTicker(roomID string) {
	uc.tickersMu.Lock()
	defer uc.tickersMu.Unlock()

	if _, ok := uc.tickers[roomID]; ok {
		return // Already running
	}

	stopChan := make(chan struct{})
	uc.tickers[roomID] = stopChan

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				ctx := context.Background()
				state, err := uc.GetState(ctx, roomID)
				if err != nil {
					log.Printf("Ticker error fetching state room %s: %v\n", roomID, err)
					continue
				}

				if state.Status != "focus" && state.Status != "break" {
					uc.stopTicker(roomID)
					return
				}

				state.RemainingSeconds--

				if state.RemainingSeconds <= 0 {
					// State transition
					if state.Status == "focus" {
						if state.CurrentCycle < state.TotalCycles {
							state.Status = "break"
							state.RemainingSeconds = state.BreakSeconds
							state.EndsAt = time.Now().Add(time.Duration(state.BreakSeconds) * time.Second).UnixNano() / int64(time.Millisecond)
							go uc.callPlaybackService(roomID, "pause")
						} else {
							state.Status = "idle"
							state.RemainingSeconds = 0
							state.CurrentCycle = 0
							state.TotalCycles = 0
							state.EndsAt = 0
							go uc.callPlaybackService(roomID, "pause")
						}
					} else if state.Status == "break" {
						if state.CurrentCycle < state.TotalCycles {
							state.Status = "focus"
							state.CurrentCycle++
							state.RemainingSeconds = state.DurationSeconds
							state.EndsAt = time.Now().Add(time.Duration(state.DurationSeconds) * time.Second).UnixNano() / int64(time.Millisecond)
							go uc.callPlaybackService(roomID, "resume")
						} else {
							state.Status = "idle"
							state.RemainingSeconds = 0
							state.CurrentCycle = 0
							state.TotalCycles = 0
							state.EndsAt = 0
							go uc.callPlaybackService(roomID, "pause")
						}
					}
				} else {
					state.EndsAt = time.Now().Add(time.Duration(state.RemainingSeconds) * time.Second).UnixNano() / int64(time.Millisecond)
				}

				if err := uc.saveState(ctx, roomID, state); err != nil {
					log.Printf("Ticker error saving state room %s: %v\n", roomID, err)
					continue
				}

				uc.broadcaster.BroadcastToRoom(roomID, "timer:sync", state)

				if state.Status == "idle" {
					uc.stopTicker(roomID)
					return
				}
			}
		}
	}()
}

func (uc *TimerUsecase) stopTicker(roomID string) {
	uc.tickersMu.Lock()
	defer uc.tickersMu.Unlock()

	if stopChan, ok := uc.tickers[roomID]; ok {
		close(stopChan)
		delete(uc.tickers, roomID)
	}
}

func (uc *TimerUsecase) callPlaybackService(roomID string, action string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := uc.playbackClient.SetPlaybackStateByTimer(ctx, &v1.SetPlaybackStateRequest{
		RoomID: roomID,
		Action: action,
	})
	if err != nil {
		log.Printf("Lỗi gọi gRPC SetPlaybackStateByTimer (%s) cho phòng %s: %v\n", action, roomID, err)
		return
	}
	if !res.Success {
		log.Printf("gRPC SetPlaybackStateByTimer (%s) cho phòng %s thất bại\n", action, roomID)
	}
}
