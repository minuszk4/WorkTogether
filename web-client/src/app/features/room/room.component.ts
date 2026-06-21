import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { ApiService } from '../../core/services/api.service';
import { StateService } from '../../core/services/state.service';
import { ChatWsService } from '../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../core/services/voice.service';
import { ToastService } from '../../shared/services/toast.service';
import { RoomUiStateService, StageMode } from './room-ui-state.service';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { StageComponent } from './components/stage/stage.component';
import { VoicePillsComponent } from './components/voice-pills/voice-pills.component';
import { ChatComponent } from './components/chat/chat.component';
import { QueueComponent } from './components/queue/queue.component';
import { PlayerBarComponent } from './components/player-bar/player-bar.component';
import { QuickReactionsComponent } from './components/quick-reactions/quick-reactions.component';
import { ReactionsCanvasComponent } from './components/reactions-canvas/reactions-canvas.component';
import { RoomVibeMeterComponent } from './components/vibe-meter/vibe-meter.component';
import { PollWidgetComponent } from './components/poll-widget/poll-widget.component';

@Component({
  selector: 'app-room',
  standalone: true,
  imports: [
    CommonModule, SidebarComponent, StageComponent,
    VoicePillsComponent, ChatComponent, QueueComponent, PlayerBarComponent,
    QuickReactionsComponent, ReactionsCanvasComponent, RoomVibeMeterComponent,
    PollWidgetComponent
  ],
  templateUrl: './room.component.html',
  styleUrl: './room.component.css'
})
export class RoomComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(ApiService);
  public state = inject(StateService);
  private chatWs = inject(ChatWsService);
  private playbackWs = inject(PlaybackWsService);
  public voiceService = inject(VoiceService);
  private toast = inject(ToastService);
  private uiState = inject(RoomUiStateService);

  public roomId = '';
  public isMicActive = false;
  public isMuted = false;
  public isScreenSharing = false;
  public isCameraActive = false;
  public isVoiceBusy = false;
  public isScreenShareBusy = false;
  public isCameraBusy = false;
  public isLeavingRoom = false;

  private subs: Subscription[] = [];
  private presenceTimer: any = null;
  private membersPollTimer: any = null;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('room_id');
    if (!id) {
      this.toast.error('Phòng không hợp lệ.');
      this.router.navigate(['/dashboard']);
      return;
    }
    this.roomId = id;
    this.loadRoomDetails();
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    this.disconnectAll();
  }

  private loadRoomDetails(): void {
    this.api.room.get(this.roomId).subscribe({
      next: (room) => {
        this.state.activeRoom$.next(room);
        this.loadRoomMembers();
        this.connectWebSockets();
        setTimeout(() => this.onVoiceChannelClick(), 800);
      },
      error: (err) => {
        this.toast.error('Không thể tải phòng: ' + err.message);
        this.router.navigate(['/dashboard']);
      }
    });
  }

  private connectWebSockets(): void {
    const token = this.state.accessToken;
    if (!token) { this.toast.error('Thiếu token.'); this.router.navigate(['/auth']); return; }
    this.chatWs.connect(this.roomId, token);
    this.playbackWs.connect(this.roomId, token);
    this.startPresenceHeartbeat();
    this.startMembersPolling();

    this.subs.push(
      this.voiceService.connected$.subscribe(c => {
        this.isMicActive = c;
        if (!c) { this.isMuted = false; this.isScreenSharing = false; this.isCameraActive = false; this.isVoiceBusy = false; }
        this.recomputeStageMode();
      }),
      this.voiceService.isMuted$.subscribe(m => (this.isMuted = m)),
      this.voiceService.isScreenSharing$.subscribe(s => { this.isScreenSharing = s; this.recomputeStageMode(); }),
      this.voiceService.isCameraActive$.subscribe(a => { this.isCameraActive = a; this.recomputeStageMode(); }),
      this.voiceService.participants$.subscribe(() => this.recomputeStageMode())
    );
  }

  private recomputeStageMode(): void {
    let mode: StageMode = 'music-only';
    if (this.voiceService.participants$.value.some(p => p.isScreenSharing)) mode = 'screenshare';
    else if (this.voiceService.participants$.value.some(p => p.isCameraOn)) mode = 'video';
    else if (this.isMicActive) mode = 'music-voice';
    this.uiState.setStageMode(mode);
  }

  public onVoiceChannelClick(): Promise<void> {
    if (this.isVoiceBusy) return Promise.resolve();
    if (this.isMicActive) {
      this.voiceService.disconnect();
      this.toast.success('Đã rời kênh voice.');
      return Promise.resolve();
    }
    this.isVoiceBusy = true;
    this.toast.info('Đang kết nối voice...');
    this.api.voice.getToken(this.roomId).subscribe({
      next: async (res) => {
        try { await this.voiceService.connect(res.livekit_url, res.token); this.toast.success('Đã tham gia voice.'); }
        catch (err: any) { this.toast.error('Lỗi voice: ' + err.message); }
        finally { this.isVoiceBusy = false; }
      },
      error: (err) => { this.isVoiceBusy = false; this.toast.error('Không xin được token voice: ' + err.message); }
    });
    return Promise.resolve();
  }

  public async onMuteToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isVoiceBusy) return;
    try {
      const target = !this.isMuted;
      await this.voiceService.setMute(target);
      this.toast.info(target ? 'Đã tắt mic.' : 'Đã bật mic.');
    } catch (err: any) { this.toast.error('Lỗi mic: ' + err.message); }
  }

  public async onScreenShareToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isScreenShareBusy) return;
    try {
      this.isScreenShareBusy = true;
      const target = !this.isScreenSharing;
      await this.voiceService.setScreenShare(target);
      this.toast.success(target ? 'Đã chia sẻ màn hình.' : 'Đã dừng chia sẻ.');
    } catch (err: any) { this.toast.error('Lỗi chia sẻ: ' + err.message); }
    finally { this.isScreenShareBusy = false; }
  }

  public async onCameraToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isCameraBusy) return;
    try {
      this.isCameraBusy = true;
      const target = !this.isCameraActive;
      await this.voiceService.setCamera(target);
      this.toast.info(target ? 'Đã bật camera.' : 'Đã tắt camera.');
    } catch (err: any) { this.toast.error('Lỗi camera: ' + err.message); }
    finally { this.isCameraBusy = false; }
  }

  public async onBackToLobbyClick(): Promise<void> {
    if (this.isLeavingRoom) return;
    this.isLeavingRoom = true;
    try { await this.api.room.leave(this.roomId).toPromise(); }
    catch (err: any) { console.warn('Leave failed:', err); this.toast.info('Đã thoát phòng.'); }
    finally {
      this.disconnectAll();
      this.router.navigate(['/dashboard']);
      this.isLeavingRoom = false;
    }
  }

  private async loadRoomMembers(): Promise<void> {
    try {
      const members = await this.api.room.listMembers(this.roomId).toPromise();
      const enrichedMembers = await Promise.all(
        (members || []).map(async (member: any) => {
          try {
            const profile = await this.api.user.getProfile(member.user_id).toPromise();
            return {
              id: member.user_id,
              member_id: member.id,
              user_id: member.user_id,
              role: member.role_type,
              permissions: member.permissions || [],
              joined_at: member.joined_at,
              display_name: profile?.display_name || profile?.username || member.user_id,
              username: profile?.username || member.user_id,
              avatar_url: profile?.avatar_url || '',
              presence: profile?.presence || { status: 'offline', custom_text: '' },
              isCurrentUser: member.user_id === this.state.user?.id
            };
          } catch {
            return {
              id: member.user_id,
              member_id: member.id,
              user_id: member.user_id,
              role: member.role_type,
              permissions: member.permissions || [],
              joined_at: member.joined_at,
              display_name: member.user_id,
              username: member.user_id,
              avatar_url: '',
              presence: { status: 'offline', custom_text: '' },
              isCurrentUser: member.user_id === this.state.user?.id
            };
          }
        })
      );

      this.state.activeRoomMembers$.next(enrichedMembers);

      const currentMember = enrichedMembers.find(member => member.isCurrentUser);
      if (currentMember) {
        this.state.roomMemberRole$.next(currentMember.role || 'MEMBER');
        this.state.roomPermissions$.next(currentMember.permissions || []);
      }
    } catch (err) {
      console.warn('Khong the tai danh sach thanh vien phong:', err);
    }
  }

  private startPresenceHeartbeat(): void {
    this.stopPresenceHeartbeat();
    
    void this.api.user.updateStatus('online', 'Xem chung').toPromise().catch(err => {
      console.warn('Initial presence update failed:', err);
    });

    this.presenceTimer = setInterval(() => {
      void this.api.user.updateStatus('online', 'Xem chung').toPromise().catch(err => {
        console.warn('Presence heartbeat update failed:', err);
      });
    }, 45 * 1000);
  }

  private stopPresenceHeartbeat(): void {
    if (this.presenceTimer) {
      clearInterval(this.presenceTimer);
      this.presenceTimer = null;
    }
  }

  private startMembersPolling(): void {
    this.stopMembersPolling();
    this.membersPollTimer = setInterval(() => {
      this.loadRoomMembers();
    }, 20000);
  }

  private stopMembersPolling(): void {
    if (this.membersPollTimer) {
      clearInterval(this.membersPollTimer);
      this.membersPollTimer = null;
    }
  }

  private disconnectAll(): void {
    this.stopPresenceHeartbeat();
    this.stopMembersPolling();
    this.chatWs.disconnect();
    this.playbackWs.disconnect();
    this.voiceService.disconnect();
    this.state.activeRoom$.next(null);
    this.state.activeRoomMembers$.next([]);
  }

  get currentUserAvatar(): string { return this.state.user?.avatar_url || ''; }
  get currentUserDisplayName(): string {
    const u = this.state.user;
    return u?.display_name || u?.username || 'WT';
  }
  get isChatOpen(): boolean { return this.uiState.uiState.isChatOpen; }
  get isQueueOpen(): boolean { return this.uiState.uiState.isQueueOpen; }
}
