package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/worktogether/services/room-service/internal/domain"
	"github.com/worktogether/services/room-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRoomNotFound         = errors.New("không tìm thấy phòng")
	ErrBannedFromRoom       = errors.New("bạn đã bị cấm khỏi phòng này")
	ErrIncorrectPassword    = errors.New("mật khẩu phòng không chính xác")
	ErrUnauthorized         = errors.New("bạn không có quyền thực hiện hành động này")
	ErrAlreadyMember        = errors.New("bạn đã là thành viên của phòng này")
	ErrNotMember            = errors.New("bạn không phải là thành viên của phòng này")
	ErrCannotKickOwner      = errors.New("không thể kick chủ phòng")
	ErrRoleNotFound         = errors.New("không tìm thấy vai trò")
	ErrTargetMemberNotFound = errors.New("thành viên mục tiêu không tồn tại")
	ErrCannotMuteOwner      = errors.New("không thể mute chủ phòng")
	ErrInvalidRoomMode      = errors.New("chế độ phòng không hợp lệ")
)

type RoomUsecase struct {
	repo *repository.PostgresRepository
	rdb  *redis.Client
}

func NewRoomUsecase(repo *repository.PostgresRepository, rdb *redis.Client) *RoomUsecase {
	return &RoomUsecase{repo: repo, rdb: rdb}
}

func (u *RoomUsecase) CreateRoom(ctx context.Context, ownerID string, req *domain.CreateRoomRequest) (*domain.Room, error) {
	var passwordHash string
	if req.Privacy == "private" && req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		passwordHash = string(hashed)
	}

	rm := &domain.Room{
		Name:         req.Name,
		Description:  req.Description,
		Privacy:      req.Privacy,
		PasswordHash: passwordHash,
		InviteCode:   u.generateInviteCode(),
		OwnerID:      ownerID,
		Mode:         roomModeOrDefault(req.Mode),
	}

	if err := u.repo.CreateRoom(ctx, rm); err != nil {
		return nil, err
	}

	// Tự động thêm chủ phòng làm thành viên đầu tiên với vai trò OWNER
	m := &domain.RoomMember{
		RoomID:   rm.ID,
		UserID:   ownerID,
		RoleType: "OWNER",
	}

	_ = u.repo.AddMember(ctx, m)

	return rm, nil
}

func roomModeOrDefault(mode string) string {
	if mode == "" {
		return "chill"
	}
	return mode
}

func isValidRoomMode(mode string) bool {
	return mode == "chill" || mode == "focus" || mode == "collaborate"
}

func (u *RoomUsecase) GetRoomByID(ctx context.Context, id string) (*domain.Room, error) {
	cacheKey := "room:" + id

	// 1. Try to get from Redis
	if u.rdb != nil {
		cachedData, err := u.rdb.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			var room domain.Room
			if err := json.Unmarshal([]byte(cachedData), &room); err == nil {
				return &room, nil
			}
		}
	}

	// 2. Cache miss, query PostgreSQL
	room, err := u.repo.GetRoomByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, nil
	}

	// 3. Save to Redis
	if u.rdb != nil {
		jsonData, err := json.Marshal(room)
		if err == nil {
			// Cache for 5 minutes
			u.rdb.Set(ctx, cacheKey, jsonData, 5*time.Minute)
		}
	}

	return room, nil
}

func (u *RoomUsecase) GetRoomByInviteCode(ctx context.Context, code string) (*domain.Room, error) {
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	if normalizedCode == "" {
		return nil, nil
	}

	return u.repo.GetRoomByInviteCode(ctx, normalizedCode)
}

func (u *RoomUsecase) GetRooms(ctx context.Context, search string, limit, offset int) ([]*domain.Room, error) {
	return u.repo.GetRooms(ctx, search, limit, offset)
}

func (u *RoomUsecase) ListMembers(ctx context.Context, requesterID, roomID string) ([]*domain.RoomMember, error) {
	member, err := u.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	return u.repo.ListMembers(ctx, roomID)
}

func (u *RoomUsecase) JoinRoom(ctx context.Context, userID, roomID, password string) (*domain.RoomMember, error) {
	// 1. Kiểm tra ban
	banned, err := u.repo.IsBanned(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBannedFromRoom
	}

	// 2. Kiểm tra đã là thành viên chưa
	existing, _ := u.repo.GetMember(ctx, roomID, userID)
	if existing != nil {
		return existing, nil
	}

	// 3. Lấy thông tin phòng
	rm, err := u.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if rm == nil {
		return nil, ErrRoomNotFound
	}

	// 4. Validate mật khẩu nếu là phòng Private
	if rm.Privacy == "private" {
		if rm.PasswordHash != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(rm.PasswordHash), []byte(password)); err != nil {
				return nil, ErrIncorrectPassword
			}
		}
	}

	// 5. Thêm thành viên
	m := &domain.RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		RoleType: "MEMBER",
	}

	err = u.repo.AddMember(ctx, m)
	if err != nil {
		return nil, err
	}

	u.invalidateRoomCache(ctx, roomID)
	return m, nil
}

func (u *RoomUsecase) LeaveRoom(ctx context.Context, userID, roomID string) error {
	m, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrNotMember
	}

	if m.RoleType == "OWNER" {
		err := u.repo.DeleteRoom(ctx, roomID)
		if err == nil { u.invalidateRoomCache(ctx, roomID) }
		return err
	}

	err = u.repo.RemoveMember(ctx, roomID, userID)
	if err == nil { u.invalidateRoomCache(ctx, roomID) }
	return err
}

func (u *RoomUsecase) CreateRole(ctx context.Context, userID, roomID string, req *domain.CreateRoleRequest) (*domain.RoomRole, error) {
	// Kiểm tra xem user có phải OWNER của phòng không
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil || member == nil || member.RoleType != "OWNER" {
		return nil, ErrUnauthorized
	}

	role := &domain.RoomRole{
		RoomID:             roomID,
		Name:               req.Name,
		CanChat:            req.CanChat,
		CanManagePlaylist:  req.CanManagePlaylist,
		CanControlPlayback: req.CanControlPlayback,
		CanModerateMembers: req.CanModerateMembers,
		CanUseVoice:        req.CanUseVoice,
	}

	err = u.repo.CreateRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (u *RoomUsecase) AssignRole(ctx context.Context, userID, roomID, targetUserID, roleID string) error {
	// Chỉ OWNER mới được gán quyền
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil || member == nil || member.RoleType != "OWNER" {
		return ErrUnauthorized
	}

	// Kiểm tra roleID có thuộc roomID không
	role, err := u.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil || role.RoomID != roomID {
		return ErrRoleNotFound
	}

	err = u.repo.UpdateMemberRole(ctx, roomID, targetUserID, roleID, "MEMBER")
	if err == nil { u.invalidateRoomCache(ctx, roomID) }
	return err
}

func (u *RoomUsecase) KickMember(ctx context.Context, operatorID, roomID, targetUserID string) error {
	// Check quyền của operator
	op, err := u.repo.GetMember(ctx, roomID, operatorID)
	if err != nil || op == nil {
		return ErrUnauthorized
	}

	allowed := op.RoleType == "OWNER" || op.RoleType == "MODERATOR"
	if !allowed && op.RoleID != "" {
		// Check custom role
		role, err := u.repo.GetRoleByID(ctx, op.RoleID)
		if err == nil && role != nil && role.CanModerateMembers {
			allowed = true
		}
	}

	if !allowed {
		return ErrUnauthorized
	}

	// Check target member
	target, err := u.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil || target == nil {
		return ErrRoomNotFound
	}

	if target.RoleType == "OWNER" {
		return ErrCannotKickOwner
	}

	err = u.repo.RemoveMember(ctx, roomID, targetUserID)
	if err == nil { u.invalidateRoomCache(ctx, roomID) }
	return err
}

func (u *RoomUsecase) BanMember(ctx context.Context, operatorID, roomID, targetUserID, reason string) error {
	// Check quyền tương tự như kick
	op, err := u.repo.GetMember(ctx, roomID, operatorID)
	if err != nil || op == nil {
		return ErrUnauthorized
	}

	allowed := op.RoleType == "OWNER" || op.RoleType == "MODERATOR"
	if !allowed && op.RoleID != "" {
		role, err := u.repo.GetRoleByID(ctx, op.RoleID)
		if err == nil && role != nil && role.CanModerateMembers {
			allowed = true
		}
	}

	if !allowed {
		return ErrUnauthorized
	}

	target, err := u.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil || target == nil {
		return ErrRoomNotFound
	}

	if target.RoleType == "OWNER" {
		return ErrCannotKickOwner
	}

	// Lưu ban
	ban := &domain.RoomBan{
		RoomID:   roomID,
		UserID:   targetUserID,
		BannedBy: operatorID,
		Reason:   reason,
	}

	if err := u.repo.AddBan(ctx, ban); err != nil {
		return err
	}

	// Trục xuất
	err = u.repo.RemoveMember(ctx, roomID, targetUserID)
	if err == nil { u.invalidateRoomCache(ctx, roomID) }
	return err
}

func (u *RoomUsecase) invalidateRoomCache(ctx context.Context, roomID string) {
	if u.rdb != nil {
		u.rdb.Del(ctx, "room:"+roomID)
	}
}

func (u *RoomUsecase) generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func (u *RoomUsecase) GetMember(ctx context.Context, roomID, userID string) (*domain.RoomMember, error) {
	return u.repo.GetMember(ctx, roomID, userID)
}

func (u *RoomUsecase) GetRoleByID(ctx context.Context, roleID string) (*domain.RoomRole, error) {
	return u.repo.GetRoleByID(ctx, roleID)
}

func (u *RoomUsecase) ResolvePermissions(ctx context.Context, member *domain.RoomMember) []string {
	if member == nil {
		return nil
	}

	permissions := []string{}
	switch member.RoleType {
	case "OWNER", "MODERATOR":
		permissions = append(permissions, "CAN_CHAT", "CAN_MANAGE_PLAYLIST", "CAN_CONTROL_PLAYBACK", "CAN_MODERATE_MEMBERS", "CAN_USE_VOICE")
	default:
		permissions = append(permissions, "CAN_CHAT", "CAN_USE_VOICE")
	}

	if member.RoleID == "" {
		return permissions
	}

	role, err := u.repo.GetRoleByID(ctx, member.RoleID)
	if err != nil || role == nil {
		return permissions
	}

	customPermissions := make([]string, 0, 5)
	if role.CanChat {
		customPermissions = append(customPermissions, "CAN_CHAT")
	}
	if role.CanManagePlaylist {
		customPermissions = append(customPermissions, "CAN_MANAGE_PLAYLIST")
	}
	if role.CanControlPlayback {
		customPermissions = append(customPermissions, "CAN_CONTROL_PLAYBACK")
	}
	if role.CanModerateMembers {
		customPermissions = append(customPermissions, "CAN_MODERATE_MEMBERS")
	}
	if role.CanUseVoice {
		customPermissions = append(customPermissions, "CAN_USE_VOICE")
	}

	return customPermissions
}

func (u *RoomUsecase) UpdateRoomSettings(ctx context.Context, userID string, roomID string, name string, description string, addMusicPolicy string, avatarURL *string, rules *string, theme string) (*domain.Room, error) {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil || member == nil || member.RoleType != "OWNER" {
		return nil, ErrUnauthorized
	}

	rm, err := u.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if rm == nil {
		return nil, ErrRoomNotFound
	}

	rm.Name = name
	rm.Description = description
	rm.AddMusicPolicy = addMusicPolicy
	rm.AvatarURL = avatarURL
	rm.Rules = rules
	rm.Theme = theme

	if err := u.repo.UpdateRoom(ctx, rm); err != nil {
		return nil, err
	}

	if u.rdb != nil {
		u.rdb.Del(ctx, "room:"+roomID)
	}

	u.invalidateRoomCache(ctx, roomID)
	return rm, nil
}

func (u *RoomUsecase) UpdateRoomMode(ctx context.Context, userID, roomID, mode string) (string, error) {
	if !isValidRoomMode(mode) {
		return "", ErrInvalidRoomMode
	}

	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return "", err
	}
	if member == nil || (member.RoleType != "OWNER" && member.RoleType != "MODERATOR") {
		return "", ErrUnauthorized
	}
	if err := u.repo.UpdateRoomMode(ctx, roomID, mode); err != nil {
		return "", err
	}

	u.invalidateRoomCache(ctx, roomID)
	if u.rdb != nil {
		message, err := json.Marshal(map[string]any{
			"event":   "room:mode_changed",
			"room_id": roomID,
			"payload": map[string]string{"mode": mode, "changed_by": userID},
		})
		if err == nil {
			u.rdb.Publish(ctx, "ch:chat:"+roomID, message)
		}
	}

	return mode, nil
}

func (u *RoomUsecase) CreateSubRoom(ctx context.Context, ownerID string, parentID string, name string, description string) (*domain.Room, error) {
	member, err := u.repo.GetMember(ctx, parentID, ownerID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.RoleType != "OWNER" {
		return nil, ErrUnauthorized
	}

	rm := &domain.Room{
		Name:         name,
		Description:  description,
		Privacy:      "public",
		InviteCode:   u.generateInviteCode(),
		OwnerID:      ownerID,
		ParentID:     &parentID,
	}

	if err := u.repo.CreateRoom(ctx, rm); err != nil {
		return nil, err
	}

	return rm, nil
}

func (u *RoomUsecase) GetSubRooms(ctx context.Context, parentID string) ([]*domain.Room, error) {
	return u.repo.GetSubRooms(ctx, parentID)
}

func (u *RoomUsecase) MoveMember(ctx context.Context, requesterID string, roomID string, userID string, subRoomID *string) error {
	if requesterID != userID {
		member, err := u.repo.GetMember(ctx, roomID, requesterID)
		if err != nil {
			return err
		}
		if member == nil || (member.RoleType != "OWNER" && member.RoleType != "MODERATOR") {
			return ErrUnauthorized
		}
	}

	if subRoomID != nil && *subRoomID != "" {
		subRoom, err := u.repo.GetRoomByID(ctx, *subRoomID)
		if err != nil {
			return err
		}
		if subRoom == nil || subRoom.ParentID == nil || *subRoom.ParentID != roomID {
			return errors.New("phòng con không hợp lệ")
		}
	}

	err := u.repo.MoveMember(ctx, roomID, userID, subRoomID)
	if err == nil { u.invalidateRoomCache(ctx, roomID); if subRoomID != nil { u.invalidateRoomCache(ctx, *subRoomID) } }
	return err
}

func (u *RoomUsecase) DeleteRoom(ctx context.Context, userID, roomID string) error {
	member, err := u.repo.GetMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if member == nil || member.RoleType != "OWNER" {
		return ErrUnauthorized
	}
	if err := u.repo.DeleteRoom(ctx, roomID); err != nil {
		return err
	}

	if u.rdb != nil {
		u.rdb.Del(ctx, "room:"+roomID)
	}

	return nil
}

func (u *RoomUsecase) MuteMember(ctx context.Context, requesterID, roomID, targetUserID string, durationSecs int) error {
	reqMember, err := u.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return err
	}
	if reqMember == nil || (reqMember.RoleType != "OWNER" && reqMember.RoleType != "MODERATOR") {
		return ErrUnauthorized
	}
	targetMember, err := u.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return err
	}
	if targetMember == nil {
		return ErrTargetMemberNotFound
	}
	if targetMember.RoleType == "OWNER" {
		return ErrCannotMuteOwner
	}
	mutedUntil := time.Now().Add(time.Duration(durationSecs) * time.Second)
	err = u.repo.UpdateMemberMute(ctx, roomID, targetUserID, &mutedUntil)
	if err == nil {
		u.invalidateRoomCache(ctx, roomID)
	}
	return err
}
func (u *RoomUsecase) UnmuteMember(ctx context.Context, requesterID, roomID, targetUserID string) error {
	reqMember, err := u.repo.GetMember(ctx, roomID, requesterID)
	if err != nil {
		return err
	}
	if reqMember == nil || (reqMember.RoleType != "OWNER" && reqMember.RoleType != "MODERATOR") {
		return ErrUnauthorized
	}
	targetMember, err := u.repo.GetMember(ctx, roomID, targetUserID)
	if err != nil {
		return err
	}
	if targetMember == nil {
		return ErrTargetMemberNotFound
	}
	err = u.repo.UpdateMemberMute(ctx, roomID, targetUserID, nil)
	if err == nil {
		u.invalidateRoomCache(ctx, roomID)
	}
	return err
}
