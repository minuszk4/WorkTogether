package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/worktogether/services/user-service/internal/domain"
	"github.com/worktogether/services/user-service/internal/repository"
)

var (
	ErrProfileNotFound       = errors.New("không tìm thấy hồ sơ người dùng")
	ErrFriendshipNotFound    = errors.New("không tìm thấy mối quan hệ bạn bè")
	ErrSelfFriendRequest     = errors.New("không thể kết bạn với chính mình")
	ErrFriendRequestNotFound = errors.New("không tìm thấy lời mời kết bạn hoặc bạn không có quyền hủy")
	ErrFriendshipNotActive   = errors.New("mối quan hệ bạn bè không tồn tại hoặc chưa được chấp nhận")
	ErrBlockNotFound         = errors.New("không tìm thấy trạng thái chặn")
	ErrCustomStatusTooLong   = errors.New("trạng thái tùy chỉnh không được quá 100 ký tự")
)

type customStatusRepository interface {
	UpdateCustomStatus(context.Context, string, string) error
}

type presenceRepository interface {
	GetPresence(context.Context, string) (*domain.Presence, error)
}

type UserUsecase struct {
	postgresRepo     *repository.PostgresRepository
	redisRepo        *repository.RedisRepository
	customStatusRepo customStatusRepository
	presenceRepo     presenceRepository
}

func NewUserUsecase(pg *repository.PostgresRepository, rdb *repository.RedisRepository) *UserUsecase {
	return &UserUsecase{
		postgresRepo:     pg,
		redisRepo:        rdb,
		customStatusRepo: pg,
		presenceRepo:     rdb,
	}
}

func (u *UserUsecase) GetProfile(ctx context.Context, id string) (*domain.UserProfile, *domain.Presence, error) {
	p, err := u.postgresRepo.GetProfileByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if p == nil {
		// Lazy-create default profile nếu chưa tồn tại
		p = &domain.UserProfile{
			ID:          id,
			DisplayName: "User_" + id[:8],
			AvatarURL:   "",
			Bio:         "Chào mừng bạn đến với WorkTogether!",
		}
		if err := u.postgresRepo.CreateProfile(ctx, p); err != nil {
			return nil, nil, err
		}
	}

	presence, err := u.redisRepo.GetPresence(ctx, id)
	if err != nil {
		// Fallback nếu Redis lỗi
		presence = &domain.Presence{
			Status:     "offline",
			CustomText: "",
			LastActive: time.Now().Unix(),
		}
	}

	return p, presence, nil
}

func (u *UserUsecase) UpdateProfile(ctx context.Context, id string, req *domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	p, err := u.postgresRepo.GetProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = &domain.UserProfile{ID: id}
	}

	p.DisplayName = req.DisplayName
	p.Bio = req.Bio
	if req.AvatarURL != "" {
		p.AvatarURL = req.AvatarURL
	}

	// Lưu thay đổi
	if err := u.postgresRepo.UpdateProfile(ctx, p); err != nil {
		// Thử create nếu chưa tồn tại
		err = u.postgresRepo.CreateProfile(ctx, p)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (u *UserUsecase) UpdatePresence(ctx context.Context, id string, req *domain.UpdateStatusRequest) (*domain.Presence, error) {
	p := &domain.Presence{
		Status:     req.Status,
		CustomText: req.CustomText,
		LastActive: time.Now().Unix(),
	}

	if err := u.redisRepo.SetPresence(ctx, id, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (u *UserUsecase) UpdateCustomStatus(ctx context.Context, userID, text string) (*domain.Presence, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > 100 {
		return nil, ErrCustomStatusTooLong
	}
	if err := u.customStatusRepo.UpdateCustomStatus(ctx, userID, text); err != nil {
		return nil, err
	}
	presence, err := u.presenceRepo.GetPresence(ctx, userID)
	if err != nil {
		return nil, err
	}
	presence.CustomText = text
	return presence, nil
}

func (u *UserUsecase) SendFriendRequest(ctx context.Context, userID, friendID string) (*domain.Friendship, error) {
	if userID == friendID {
		return nil, ErrSelfFriendRequest
	}

	// Đảm bảo cả 2 tài khoản đều tồn tại profile
	_, _, _ = u.GetProfile(ctx, userID)
	_, _, _ = u.GetProfile(ctx, friendID)

	return u.postgresRepo.CreateFriendRequest(ctx, userID, friendID)
}

func (u *UserUsecase) RespondFriendRequest(ctx context.Context, friendshipID, action string) error {
	status := "REJECTED"
	if action == "accept" {
		status = "ACCEPTED"
	}

	return u.postgresRepo.UpdateFriendshipStatus(ctx, friendshipID, status)
}

func (u *UserUsecase) GetFriends(ctx context.Context, userID, status string) ([]*domain.Friendship, error) {
	return u.postgresRepo.GetFriendships(ctx, userID, status)
}

func (u *UserUsecase) BlockUser(ctx context.Context, userID, targetID string) error {
	return u.postgresRepo.BlockUser(ctx, userID, targetID)
}

func (u *UserUsecase) CancelFriendRequest(ctx context.Context, userID, friendshipID string) error {
	err := u.postgresRepo.CancelFriendRequest(ctx, friendshipID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFriendRequestNotFound
		}
		return err
	}
	return nil
}

func (u *UserUsecase) Unfriend(ctx context.Context, userID, friendshipID string) error {
	err := u.postgresRepo.Unfriend(ctx, friendshipID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFriendshipNotActive
		}
		return err
	}
	return nil
}

func (u *UserUsecase) UnblockUser(ctx context.Context, userID, friendshipID string) error {
	err := u.postgresRepo.UnblockUser(ctx, friendshipID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBlockNotFound
		}
		return err
	}
	return nil
}
