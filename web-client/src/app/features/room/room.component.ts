import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { ApiService } from '../../core/services/api.service';
import { StateService } from '../../core/services/state.service';
import { ChatWsService } from '../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../core/services/voice.service';
import { ToastService } from '../../shared/services/toast.service';
import { RoomMode, RoomUiStateService, StageMode } from './room-ui-state.service';
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
import { SubroomsComponent } from './components/subrooms/subrooms.component';
import { TimerComponent } from './components/timer/timer.component';
import { CollabNotesComponent } from './components/collab-notes/collab-notes.component';
import { LyricsComponent } from './components/lyrics/lyrics.component';
import { BookmarksComponent } from './components/bookmarks/bookmarks.component';
import { PlayerEngineService } from './components/player-engine/player-engine.service';
import { RoomIdentityComponent } from './components/room-identity/room-identity.component';
import { HistoryStatsComponent } from './components/history-stats/history-stats.component';
import { SessionComponent } from './components/session/session.component';
import { WhiteboardComponent } from './components/whiteboard/whiteboard.component';

@Component({
  selector: 'app-room',
  standalone: true,
  imports: [
	CommonModule, FormsModule, SidebarComponent, StageComponent,
    VoicePillsComponent, ChatComponent, QueueComponent, PlayerBarComponent,
    QuickReactionsComponent, ReactionsCanvasComponent, RoomVibeMeterComponent,
    PollWidgetComponent, SubroomsComponent, TimerComponent, CollabNotesComponent,
    LyricsComponent, BookmarksComponent,
    RoomIdentityComponent, HistoryStatsComponent, SessionComponent, WhiteboardComponent
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
  private playerEngine = inject(PlayerEngineService);

  public roomId = '';
  public isMicActive = false;
  public isMuted = false;
  public isScreenSharing = false;
  public isCameraActive = false;
  public isVoiceBusy = false;
  public isScreenShareBusy = false;
  public isCameraBusy = false;
  public isLeavingRoom = false;
	public isVoicePrejoinOpen = false;
	public audioInputDevices: MediaDeviceInfo[] = [];
	public selectedAudioInput = '';
  public currentSubRoomId: string | null = null;
  public isWhiteboardOpen = false;
  public subtitleOriginal = '';
  public subtitleTranslation = '';
  public isCaptionsEnabled = false;
  private captionUploadBusy = false;
  public savedVolume: number | null = null;
  private currentSubtitleId = '';

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
        this.uiState.applyRoomMode(this.normalizeRoomMode(room.mode));
        this.loadRoomMembers();
        this.connectWebSockets();
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
    this.chatWs.connect(this.roomId, token, this.state.user?.preferred_language || '');
    this.playbackWs.connect(this.roomId, token);
    this.startPresenceHeartbeat();
    this.startMembersPolling();

    this.subs.push(
      this.voiceService.connected$.subscribe(c => {
        this.isMicActive = c;
        if (!c) { this.isMuted = false; this.isScreenSharing = false; this.isCameraActive = false; this.isVoiceBusy = false; this.isCaptionsEnabled = false; }
        this.recomputeStageMode();
      }),
      this.voiceService.isMuted$.subscribe(m => (this.isMuted = m)),
      this.voiceService.isScreenSharing$.subscribe(s => { this.isScreenSharing = s; this.recomputeStageMode(); }),
      this.voiceService.isCameraActive$.subscribe(a => { this.isCameraActive = a; this.recomputeStageMode(); }),
      this.voiceService.participants$.subscribe(() => this.recomputeStageMode()),
      this.state.activeRoomMembers$.subscribe(members => {
        const me = members.find(m => m.isCurrentUser);
        if (me) {
          const targetSubRoomId = me.active_sub_room_id || null;
          if (targetSubRoomId !== this.currentSubRoomId) {
            this.currentSubRoomId = targetSubRoomId;
            this.handleVoiceSubRoomSwitch(targetSubRoomId);
          }
        }
      }),
      this.chatWs.roomMode$.subscribe(change => {
        if (change) this.applyRoomMode(change.mode);
      }),
      this.chatWs.subtitleReceived$.subscribe(subtitle => {
        this.currentSubtitleId = subtitle.id;
        this.subtitleOriginal = subtitle.text;
        this.subtitleTranslation = '';
      }),
      this.chatWs.translationReceived$.subscribe(translation => {
        if (translation.kind === 'transcript' && translation.event_id === this.currentSubtitleId) {
          this.subtitleTranslation = translation.text;
        }
      })
    );
  }

  public changeRoomMode(mode: RoomMode): void {
    if (!this.canChangeRoomMode || mode === this.uiState.uiState.roomMode) return;
    this.api.room.updateMode(this.roomId, mode).subscribe({
      next: () => this.applyRoomMode(mode),
      error: (err) => this.toast.error('Không thể đổi chế độ phòng: ' + (err.message || 'lỗi không xác định.'))
    });
  }

  private applyRoomMode(mode: RoomMode): void {
    this.uiState.applyRoomMode(mode);
    const room = this.state.activeRoom$.value;
    if (room) this.state.activeRoom$.next({ ...room, mode });
    this.updateRoomPresence();
  }

  private normalizeRoomMode(mode: unknown): RoomMode {
    return mode === 'focus' || mode === 'collaborate' ? mode : 'chill';
  }

  private recomputeStageMode(): void {
    let mode: StageMode = 'music-only';
    if (this.voiceService.participants$.value.some(p => p.isScreenSharing)) mode = 'screenshare';
    else if (this.voiceService.participants$.value.some(p => p.isCameraOn)) mode = 'video';
    else if (this.isMicActive) mode = 'music-voice';
    this.uiState.setStageMode(mode);
  }

  public async onVoiceChannelClick(): Promise<void> {
    if (this.isVoiceBusy) return Promise.resolve();
    if (this.isMicActive) {
      this.voiceService.disconnect();
      this.toast.success('Đã rời kênh voice.');
      return Promise.resolve();
    }
	this.isVoicePrejoinOpen = true;
	try {
	  const devices = await navigator.mediaDevices?.enumerateDevices();
	  this.audioInputDevices = (devices || []).filter(device => device.kind === 'audioinput');
	  this.selectedAudioInput ||= this.audioInputDevices[0]?.deviceId || '';
	} catch {
	  this.audioInputDevices = [];
	}
	return Promise.resolve();
  }

	public confirmVoiceJoin(): void {
	this.isVoicePrejoinOpen = false;
    this.isVoiceBusy = true;
    this.toast.info('Đang kết nối voice...');
	this.api.voice.getToken(this.roomId, this.currentSubRoomId).subscribe({
      next: async (res) => {
		try { await this.voiceService.connect(res.livekit_url, res.token, this.selectedAudioInput); this.toast.success('Đã tham gia voice với mic tắt.'); }
        catch (err: any) { this.toast.error('Lỗi voice: ' + err.message); }
        finally { this.isVoiceBusy = false; }
      },
      error: (err) => { this.isVoiceBusy = false; this.toast.error('Không xin được token voice: ' + err.message); }
    });
  }

  public async onMuteToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isVoiceBusy) return;
    try {
      const target = !this.isMuted;
      await this.voiceService.setMute(target);
      this.toast.info(target ? 'Đã tắt mic.' : 'Đã bật mic.');
    } catch (err: any) { this.toast.error('Lỗi mic: ' + err.message); }
  }

  public onCaptionsToggleClick(): void {
    const enabled = !this.isCaptionsEnabled;
    const started = this.voiceService.setCaptionsEnabled(enabled, audio => this.sendCaptionAudio(audio));
    if (enabled && !started) {
      this.toast.info('Bật microphone trước khi dùng phụ đề.');
      return;
    }
    this.isCaptionsEnabled = enabled;
    this.toast.info(enabled ? 'Đã bật phụ đề giọng nói.' : 'Đã tắt phụ đề giọng nói.');
  }

  private sendCaptionAudio(audio: Blob): void {
    if (this.captionUploadBusy || audio.size > 128 * 1024) return;
    this.captionUploadBusy = true;
    // ponytail: one request at a time; increase concurrency only if captions demonstrably fall behind.
    this.api.chat.transcribeCaption(this.roomId, audio, navigator.language || 'en-US').subscribe({
      error: (error) => {
        this.captionUploadBusy = false;
        if (error?.status === 503) {
          this.voiceService.setCaptionsEnabled(false);
          this.isCaptionsEnabled = false;
          this.toast.error('Phụ đề chưa được cấu hình trên server.');
        }
      },
      complete: () => { this.captionUploadBusy = false; }
    });
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
            const profile = await this.api.user.getProfile(member.user_id, this.roomId).toPromise();
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
    this.updateRoomPresence();
    this.presenceTimer = setInterval(() => {
      this.updateRoomPresence();
    }, 45 * 1000);
  }

  private updateRoomPresence(): void {
    void this.api.user.heartbeatPresence('online').toPromise().catch(err => {
      console.warn('Room presence update failed:', err);
    });
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
    }, 4000);
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
	get voiceConnectionState(): string { return this.voiceService.connectionState$.value; }
  get currentUserDisplayName(): string {
    const u = this.state.user;
    return u?.display_name || u?.username || 'WT';
  }
  get isChatOpen(): boolean { return this.uiState.uiState.isChatOpen; }
  get isQueueOpen(): boolean { return this.uiState.uiState.isQueueOpen; }
  get isSubroomsOpen(): boolean { return this.uiState.uiState.isSubroomsOpen; }
  get isNotesOpen(): boolean { return this.uiState.uiState.isNotesOpen; }
  get isTimerOpen(): boolean { return this.uiState.uiState.isTimerOpen; }
  get isLyricsOpen(): boolean { return this.uiState.uiState.isLyricsOpen; }
  get isBookmarksOpen(): boolean { return this.uiState.uiState.isBookmarksOpen; }
  get isIdentityOpen(): boolean { return this.uiState.uiState.isIdentityOpen; }
  get isStatsOpen(): boolean { return this.uiState.uiState.isStatsOpen; }
  get roomMode(): RoomMode { return this.uiState.uiState.roomMode; }
  get canChangeRoomMode(): boolean {
    const role = this.state.roomMemberRole$.value;
    return role === 'OWNER' || role === 'MODERATOR';
  }

  private handleVoiceSubRoomSwitch(subRoomId: string | null): void {
	const wasConnected = this.voiceService.connected$.value;
    this.voiceService.disconnect();
    this.isMicActive = false;
	if (!wasConnected) return;
    
    if (subRoomId) {
      // Mute main music player when entering breakout subroom
      if (this.savedVolume === null) {
        this.savedVolume = this.playerEngine.volume;
      }
      this.playerEngine.setVolumeFromFraction(0);

      this.toast.info('Đang kết nối voice phòng con...');
	  this.api.voice.getToken(this.roomId, subRoomId).subscribe({
        next: async (res) => {
          try {
            await this.voiceService.connect(res.livekit_url, res.token);
            this.toast.success('Đã vào thảo luận nhóm con.');
          } catch (err: any) {
            this.toast.error('Lỗi voice phòng con: ' + err.message);
          }
        },
        error: (err) => {
          this.toast.error('Không xin được token voice phòng con: ' + err.message);
        }
      });
    } else {
      // Restore main music player volume when returning to lobby
      if (this.savedVolume !== null) {
        this.playerEngine.setVolumeFromFraction(this.savedVolume / 100);
        this.savedVolume = null;
      }

      this.toast.info('Đang quay lại voice sảnh chính...');
	  this.api.voice.getToken(this.roomId, null).subscribe({
        next: async (res) => {
          try {
            await this.voiceService.connect(res.livekit_url, res.token);
            this.toast.success('Đã quay lại voice sảnh chính.');
          } catch (err: any) {
            this.toast.error('Lỗi voice sảnh chính: ' + err.message);
          }
        },
        error: (err) => {
          this.toast.error('Không xin được token voice sảnh chính: ' + err.message);
        }
      });
    }
  }
}
