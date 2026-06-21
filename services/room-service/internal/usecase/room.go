package usecase

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/worktogether/services/room-service/internal/domain"
	"github.com/worktogether/services/room-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRoomNotFound      = errors.New("không tìm thấy phòng")
	ErrBannedFromRoom    = errors.New("bạn đã bị cấm khỏi phòng này")
	ErrIncorrectPassword = errors.New("mật khẩu phòng không chính xác")
	ErrUnauthorized      = errors.New("bạn không có quyền thực hiện hành động này")
	ErrAlreadyMember     = errors.New("bạn đã là thành viên của phòng này")
	ErrNotMember         = errors.New("bạn không phải là thành viên của phòng này")
	ErrCannotKickOwner   = errors.New("không thể kick chủ phòng")
	ErrRoleNotFound      = errors.New("không tìm thấy vai trò")
)

type RoomUsecase struct {
	repo *repository.PostgresRepository
}

func NewRoomUsecase(repo *repository.PostgresRepository) *RoomUsecase {
	return &RoomUsecase{repo: repo}
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

func (u *RoomUsecase) GetRoomByID(ctx context.Context, id string) (*domain.Room, error) {
	return u.repo.GetRoomByID(ctx, id)
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
		// Trong MVP: Nếu chủ phòng rời đi, xóa phòng
		return u.repo.DeleteRoom(ctx, roomID)
	}

	return u.repo.RemoveMember(ctx, roomID, userID)
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

	return u.repo.UpdateMemberRole(ctx, roomID, targetUserID, roleID, "MEMBER")
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

	return u.repo.RemoveMember(ctx, roomID, targetUserID)
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
	return u.repo.RemoveMember(ctx, roomID, targetUserID)
}

func (u *RoomUsecase) generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
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

func (u *RoomUsecase) UpdateRoomSettings(ctx context.Context, userID string, roomID string, name string, description string, addMusicPolicy string) (*domain.Room, error) {
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

	if err := u.repo.UpdateRoom(ctx, rm); err != nil {
		return nil, err
	}

	return rm, nil
}
