import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { ApiService } from '../../core/services/api.service';
import { NotificationStreamService } from '../../core/services/notification-stream.service';
import { StateService } from '../../core/services/state.service';
import { ToastService } from '../../shared/services/toast.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.css'
})
export class DashboardComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  public state = inject(StateService);
  private toast = inject(ToastService);
  private router = inject(Router);
  private notificationStream = inject(NotificationStreamService);

  private presenceTimer: ReturnType<typeof setInterval> | null = null;
  private profileCache = new Map<string, Promise<any>>();
  private hasLiveNotificationConnection = false;

  public isCreateRoomModalOpen = false;
  public isProfileSettingsModalOpen = false;
  public isNotificationPanelOpen = false;
  public isRoomLoading = true;
  public isFriendsLoading = true;
  public isPendingRequestsLoading = true;
  public isNotificationsLoading = true;
  public isInviteJoining = false;
  public isAvatarUploading = false;

  public roomName = '';
  public roomDesc = '';
  public roomPrivacy = 'public';
  public roomPassword = '';
  public roomMode: 'chill' | 'focus' | 'collaborate' = 'chill';

  public profileDisplayName = '';
  public profileBio = '';
  public profileAvatarUrl = '';

  public addFriendId = '';
  public inviteCode = '';
  public presenceStatus = 'online';

  public notifications: any[] = [];
  public pendingFriendRequests: any[] = [];
  public actionItems: any[] = [];
  public isActionItemsLoading = true;
  public sessionRecaps: any[] = [];
  public isSessionRecapsLoading = true;
  public selectedSessionRecap: any | null = null;
  public isSessionRecapLoading = false;

  ngOnInit(): void {
    const user = this.state.user;
    if (!user) {
      this.router.navigate(['/auth']);
      return;
    }

    this.profileDisplayName = user.display_name || user.username;
    this.profileBio = user.bio || '';
    this.profileAvatarUrl = user.avatar_url || '';
    this.applyDashboardLoadState();

    void this.initializeDashboard();
  }

  ngOnDestroy(): void {
    this.stopPresenceHeartbeat();
    this.hasLiveNotificationConnection = false;
    this.notificationStream.disconnect();
  }

  public get roomCount(): number {
    return this.state.roomList.length;
  }

  public get friendCount(): number {
    return this.state.friends.length;
  }

  public get unreadNotificationCount(): number {
    return this.state.unreadNotificationsCount;
  }

  public get visibleNotifications(): any[] {
    return this.notifications.slice(0, 8);
  }

  public get isAdmin(): boolean {
    try {
      const token = this.state.accessToken;
      const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
      return payload.is_admin === true;
    } catch {
      return false;
    }
  }

  public openAdmin(): void { void this.router.navigate(['/admin']); }

  public get openActionItems(): any[] {
    return this.actionItems.filter(item => item.status === 'OPEN');
  }

  public async loadActionItems(): Promise<void> {
    this.isActionItemsLoading = true;
    try {
      this.actionItems = await firstValueFrom(this.api.room.listMyActionItems());
    } catch (error: any) {
      this.toast.error('Khong the tai action items: ' + this.errorMessage(error));
    } finally {
      this.isActionItemsLoading = false;
    }
  }

  public async toggleActionItem(item: any): Promise<void> {
    const status = item.status === 'DONE' ? 'OPEN' : 'DONE';
    try {
      await firstValueFrom(this.api.room.updateActionItem(item.room_id, item.session_id, item.id, status));
      item.status = status;
    } catch (error: any) {
      this.toast.error('Khong the cap nhat action item: ' + this.errorMessage(error));
    }
  }

  public async loadSessionRecaps(): Promise<void> {
    this.isSessionRecapsLoading = true;
    try {
      this.sessionRecaps = await firstValueFrom(this.api.room.listMySessionRecaps());
    } catch (error: any) {
      this.toast.error('Khong the tai session da hoan tat: ' + this.errorMessage(error));
    } finally {
      this.isSessionRecapsLoading = false;
    }
  }

  public async openSessionRecap(recap: any): Promise<void> {
    this.isSessionRecapLoading = true;
    this.selectedSessionRecap = { ...recap, workspace: null };
    try {
      const workspace = await firstValueFrom(this.api.room.getSessionWorkspace(recap.room_id, recap.id));
      this.selectedSessionRecap = { ...recap, workspace };
    } catch (error: any) {
      this.selectedSessionRecap = null;
      this.toast.error('Khong the tai recap: ' + this.errorMessage(error));
    } finally {
      this.isSessionRecapLoading = false;
    }
  }

  public async loadRooms(): Promise<void> {
    this.isRoomLoading = !this.state.dashboardSectionState.roomsReady;

    try {
      const rooms = await firstValueFrom(this.api.room.list());
      this.state.roomList$.next(rooms || []);
    } catch (error: any) {
      this.toast.error('Khong the tai danh sach phong: ' + this.errorMessage(error));
    } finally {
      this.isRoomLoading = false;
      this.state.setDashboardSectionReady('roomsReady');
    }
  }

  public async loadFriends(): Promise<void> {
    this.isFriendsLoading = !this.state.dashboardSectionState.friendsReady;

    try {
      const friends = await firstValueFrom(this.api.user.getFriends());
      const normalizedFriends = await Promise.all((friends || []).map((friend) => this.normalizeFriendship(friend)));
      this.state.friends$.next(normalizedFriends);
    } catch (error: any) {
      this.toast.error('Khong the tai danh sach ban be: ' + this.errorMessage(error));
    } finally {
      this.isFriendsLoading = false;
      this.state.setDashboardSectionReady('friendsReady');
    }
  }

  public async loadPendingFriendRequests(): Promise<void> {
    this.isPendingRequestsLoading = !this.state.dashboardSectionState.pendingReady;

    try {
      const requests = await firstValueFrom(this.api.user.getPendingFriends());
      const normalizedRequests = await Promise.all((requests || []).map((request) => this.normalizeFriendship(request)));
      this.pendingFriendRequests = normalizedRequests.filter((request) => request.isIncoming);
    } catch (error: any) {
      this.toast.error('Khong the tai loi moi ket ban: ' + this.errorMessage(error));
    } finally {
      this.isPendingRequestsLoading = false;
      this.state.setDashboardSectionReady('pendingReady');
    }
  }

  public async loadNotifications(): Promise<void> {
    this.isNotificationsLoading = !this.state.dashboardSectionState.notificationsReady;

    try {
      const [notifications, unreadSummary] = await Promise.all([
        firstValueFrom(this.api.notification.list()),
        firstValueFrom(this.api.notification.unreadCount())
      ]);

      this.notifications = (notifications || []).map((item) => this.normalizeNotification(item));
      this.state.notifications$.next(this.notifications);
      this.state.unreadNotificationsCount$.next(Number(unreadSummary?.unread_count || 0));
    } catch (error: any) {
      this.toast.error('Khong the tai notifications: ' + this.errorMessage(error));
    } finally {
      this.isNotificationsLoading = false;
      this.state.setDashboardSectionReady('notificationsReady');
    }
  }

  public async togglePresence(): Promise<void> {
    const statuses = ['online', 'away', 'busy', 'offline'];
    const currentIndex = statuses.indexOf(this.presenceStatus);
    const nextStatus = statuses[(currentIndex + 1) % statuses.length];

    try {
      await firstValueFrom(this.api.user.updateStatus(nextStatus, ''));
      this.presenceStatus = nextStatus;
      this.toast.info(`Da doi trang thai sang ${nextStatus.toUpperCase()}`);
    } catch {
      this.toast.error('Khong the cap nhat trang thai hien tai.');
    }
  }

  public async onCreateRoomSubmit(): Promise<void> {
    if (!this.roomName.trim()) {
      this.toast.error('Ten phong khong duoc de trong.');
      return;
    }

    try {
      const effectivePassword = this.roomPrivacy === 'private' ? this.roomPassword : '';
      const room = await firstValueFrom(
        this.api.room.create(
          this.roomName.trim(),
          this.roomDesc.trim(),
          this.roomPrivacy,
          effectivePassword,
          this.roomMode
        )
      );

      this.isCreateRoomModalOpen = false;
      this.roomName = '';
      this.roomDesc = '';
      this.roomPrivacy = 'public';
      this.roomPassword = '';
      this.roomMode = 'chill';

      await this.loadRooms();
      this.toast.success(`Da tao phong "${room.name}". Invite code: ${room.invite_code}`);
      await this.joinRoom(room.id);
    } catch (error: any) {
      this.toast.error(this.errorMessage(error) || 'Khong the tao phong.');
    }
  }

  public async joinRoom(roomId: string, password = ''): Promise<void> {
    try {
      const joinData = await firstValueFrom(this.api.room.join(roomId, password));
      this.applyRoomAccess(joinData);

      const roomDetails = await firstValueFrom(this.api.room.get(roomId));
      this.state.activeRoom$.next(roomDetails);

      this.stopPresenceHeartbeat();
      this.hasLiveNotificationConnection = false;
      this.notificationStream.disconnect();
      await this.router.navigate(['/room', roomId]);
    } catch (error: any) {
      const message = this.errorMessage(error);

      if (message.includes('mat khau') || message.includes('PASSWORD')) {
        const nextPassword = window.prompt('Nhap mat khau phong:');
        if (nextPassword !== null) {
          await this.joinRoom(roomId, nextPassword);
        }
        return;
      }

      if (message.includes('ALREADY_MEMBER') || message.includes('thanh vien')) {
        try {
          const roomDetails = await firstValueFrom(this.api.room.get(roomId));
          this.state.activeRoom$.next(roomDetails);
          this.stopPresenceHeartbeat();
          this.hasLiveNotificationConnection = false;
          this.notificationStream.disconnect();
          await this.router.navigate(['/room', roomId]);
          return;
        } catch {
          this.toast.error('Khong the tai thong tin phong.');
          return;
        }
      }

      this.toast.error('Khong the tham gia phong: ' + message);
    }
  }

  public async joinRoomByInviteCode(): Promise<void> {
    const normalizedCode = this.inviteCode.trim().toUpperCase();
    if (!normalizedCode) {
      this.toast.error('Nhap invite code truoc khi vao phong.');
      return;
    }

    this.isInviteJoining = true;
    try {
      const room = await firstValueFrom(this.api.room.getByInviteCode(normalizedCode));
      this.inviteCode = '';
      await this.joinRoom(room.id);
    } catch (error: any) {
      this.toast.error('Khong tim thay phong tu invite code: ' + this.errorMessage(error));
    } finally {
      this.isInviteJoining = false;
    }
  }

  public async onAddFriendSubmit(): Promise<void> {
    const friendId = this.addFriendId.trim();
    if (!friendId) {
      return;
    }

    try {
      await firstValueFrom(this.api.user.sendFriendRequest(friendId));
      this.addFriendId = '';
      this.toast.success('Da gui loi moi ket ban.');
      await Promise.allSettled([
        this.loadPendingFriendRequests(),
        this.loadNotifications()
      ]);
    } catch (error: any) {
      this.toast.error(this.errorMessage(error) || 'Khong the gui loi moi ket ban.');
    }
  }

  public async respondToFriendRequest(friendshipId: string, action: 'accept' | 'reject'): Promise<void> {
    try {
      await firstValueFrom(this.api.user.respondFriendRequest(friendshipId, action));
      this.toast.success(action === 'accept' ? 'Da chap nhan loi moi ket ban.' : 'Da tu choi loi moi ket ban.');
      await Promise.allSettled([
        this.loadPendingFriendRequests(),
        this.loadFriends(),
        this.loadNotifications()
      ]);
    } catch (error: any) {
      this.toast.error('Khong the xu ly loi moi ket ban: ' + this.errorMessage(error));
    }
  }

  public async markNotificationRead(notification: any, event?: MouseEvent): Promise<void> {
    event?.stopPropagation();
    if (notification.is_read) {
      return;
    }

    try {
      await firstValueFrom(this.api.notification.markRead(notification.id));
      this.notifications = this.notifications.map((item) =>
        item.id === notification.id ? { ...item, is_read: true } : item
      );
      this.state.notifications$.next(this.notifications);
      this.state.unreadNotificationsCount$.next(Math.max(0, this.state.unreadNotificationsCount - 1));
    } catch (error: any) {
      this.toast.error('Khong the danh dau notification da doc: ' + this.errorMessage(error));
    }
  }

  public async markAllNotificationsRead(): Promise<void> {
    try {
      await firstValueFrom(this.api.notification.markAllRead());
      this.notifications = this.notifications.map((item) => ({ ...item, is_read: true }));
      this.state.notifications$.next(this.notifications);
      this.state.unreadNotificationsCount$.next(0);
    } catch (error: any) {
      this.toast.error('Khong the danh dau tat ca notifications: ' + this.errorMessage(error));
    }
  }

  public toggleNotificationPanel(): void {
    this.isNotificationPanelOpen = !this.isNotificationPanelOpen;
  }

  public notificationTypeLabel(notification: any): string {
    const type = String(notification?.type || '').toUpperCase();
    switch (type) {
      case 'FRIEND_REQUEST':
        return 'Friend';
      case 'CHAT_MENTION':
        return 'Mention';
      case 'ROOM_INVITE':
        return 'Invite';
      default:
        return 'Notice';
    }
  }

  public notificationTypeClass(notification: any): string {
    const type = String(notification?.type || '').toLowerCase();
    return `tone-${type || 'system'}`;
  }

  public userLabel(profile: any): string {
    const base = profile?.display_name || profile?.username || 'WT';
    return String(base).slice(0, 2).toUpperCase();
  }

  public async onAvatarFileSelected(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) {
      return;
    }

    this.isAvatarUploading = true;
    const formData = new FormData();
    formData.append('file', file);
    formData.append('title', `avatar-${this.state.user?.id}`);
    formData.append('artist', this.state.user?.username || 'user');

    try {
      this.toast.info('Dang tai anh dai dien...');
      const trackData = await firstValueFrom(this.api.music.upload(formData));
      this.profileAvatarUrl = trackData.source_url;
      this.toast.success('Da cap nhat anh dai dien tam thoi.');
    } catch (error: any) {
      this.toast.error('Tai anh that bai: ' + this.errorMessage(error));
    } finally {
      this.isAvatarUploading = false;
      input.value = '';
    }
  }

  public async onSaveProfileSubmit(): Promise<void> {
    try {
      const updated = await firstValueFrom(
        this.api.user.updateProfile(
          this.profileDisplayName.trim(),
          this.profileBio.trim(),
          this.profileAvatarUrl
        )
      );

      this.state.user$.next(updated);
      this.isProfileSettingsModalOpen = false;
      this.toast.success('Da luu cau hinh ho so.');
    } catch (error: any) {
      this.toast.error('Luu ho so that bai: ' + this.errorMessage(error));
    }
  }

  public async logout(): Promise<void> {
    try {
      await firstValueFrom(this.api.auth.logout());
      this.hasLiveNotificationConnection = false;
      this.notificationStream.disconnect();
      this.stopPresenceHeartbeat();
      this.state.clear();
      await this.router.navigate(['/auth']);
      this.toast.success('Dang xuat thanh cong.');
    } catch {
      this.toast.error('Loi dang xuat.');
    }
  }

  private async initializeDashboard(): Promise<void> {
    await Promise.allSettled([
      this.loadRooms(),
      this.loadFriends(),
      this.loadPendingFriendRequests(),
      this.loadNotifications(),
      this.loadActionItems(),
      this.loadSessionRecaps()
    ]);

    this.connectNotificationStream();
    this.startPresenceHeartbeat();
  }

  private connectNotificationStream(): void {
    const token = this.state.accessToken;
    if (!token || this.hasLiveNotificationConnection) {
      return;
    }

    this.hasLiveNotificationConnection = true;
    this.notificationStream.connect(token, (payload) => {
      const notification = this.normalizeNotification(payload);
      const existing = this.notifications.find((item) => item.id === notification.id);
      const nextNotifications = [
        notification,
        ...this.notifications.filter((item) => item.id !== notification.id)
      ].slice(0, 50);

      this.notifications = nextNotifications;
      this.state.notifications$.next(nextNotifications);

      if (!existing && !notification.is_read) {
        this.state.unreadNotificationsCount$.next(this.state.unreadNotificationsCount + 1);
      }

      if (notification.type === 'FRIEND_REQUEST') {
        void this.loadPendingFriendRequests();
      }
    });
  }

  private startPresenceHeartbeat(): void {
    this.stopPresenceHeartbeat();
    this.presenceTimer = setInterval(() => {
      void firstValueFrom(this.api.user.updateStatus(this.presenceStatus, 'Lobby active')).catch((error) => {
        console.warn('Presence heartbeat update failed:', error);
      });
    }, 45 * 1000);
  }

  private stopPresenceHeartbeat(): void {
    if (this.presenceTimer) {
      clearInterval(this.presenceTimer);
      this.presenceTimer = null;
    }
  }

  private applyDashboardLoadState(): void {
    const cache = this.state.dashboardSectionState;
    this.isRoomLoading = !cache.roomsReady;
    this.isFriendsLoading = !cache.friendsReady;
    this.isPendingRequestsLoading = !cache.pendingReady;
    this.isNotificationsLoading = !cache.notificationsReady;
  }

  private async normalizeFriendship(friendship: any): Promise<any> {
    const currentUserId = this.state.user?.id || '';
    const friendUserId = friendship?.user_id === currentUserId
      ? friendship?.friend_id
      : friendship?.user_id;
    const profile = await this.getUserProfile(friendUserId);

    return {
      id: friendship?.id || `${friendUserId}-${friendship?.status || 'friend'}`,
      friendship_id: friendship?.id || '',
      friend_id: friendUserId,
      user_id: friendship?.user_id || '',
      status: friendship?.status || 'UNKNOWN',
      created_at: friendship?.created_at,
      updated_at: friendship?.updated_at,
      isIncoming: friendship?.friend_id === currentUserId,
      friend_profile: {
        id: friendUserId,
        username: profile?.username || friendUserId,
        display_name: profile?.display_name || profile?.username || `User_${String(friendUserId || '').slice(0, 8)}`,
        avatar_url: profile?.avatar_url || ''
      },
      presence: profile?.presence || { status: 'offline', custom_text: '' }
    };
  }

  private async getUserProfile(userId: string): Promise<any> {
    if (!userId) {
      return null;
    }

    if (!this.profileCache.has(userId)) {
      this.profileCache.set(
        userId,
        firstValueFrom(this.api.user.getProfile(userId)).catch(() => ({
          id: userId,
          username: userId,
          display_name: `User_${String(userId).slice(0, 8)}`,
          avatar_url: '',
          presence: { status: 'offline', custom_text: '' }
        }))
      );
    }

    return this.profileCache.get(userId) as Promise<any>;
  }

  private applyRoomAccess(joinData: any): void {
    const role = joinData?.role || joinData?.role_type || 'MEMBER';
    const permissions = Array.isArray(joinData?.permissions)
      ? joinData.permissions
      : this.defaultPermissionsForRole(role);

    this.state.roomMemberRole$.next(role);
    this.state.roomPermissions$.next(permissions);
  }

  private defaultPermissionsForRole(role: string): string[] {
    if (role === 'OWNER' || role === 'MODERATOR') {
      return ['CAN_CHAT', 'CAN_MANAGE_PLAYLIST', 'CAN_CONTROL_PLAYBACK', 'CAN_MODERATE_MEMBERS', 'CAN_USE_VOICE'];
    }

    return ['CAN_CHAT', 'CAN_USE_VOICE'];
  }

  private normalizeNotification(notification: any): any {
    const type = String(notification?.type || 'SYSTEM').toUpperCase();
    return {
      id: notification?.id || `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      type,
      title: notification?.title || this.defaultNotificationTitle(type),
      content: notification?.content || '',
      is_read: Boolean(notification?.is_read),
      sender_id: notification?.sender_id || '',
      created_at: notification?.created_at || new Date().toISOString()
    };
  }

  private defaultNotificationTitle(type: string): string {
    switch (type) {
      case 'FRIEND_REQUEST':
        return 'Friend request';
      case 'CHAT_MENTION':
        return 'Chat mention';
      case 'ROOM_INVITE':
        return 'Room invite';
      default:
        return 'System update';
    }
  }

  private errorMessage(error: any): string {
    const rawMessage = String(error?.message || 'Server error');
    return rawMessage.trim();
  }
}
