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
import { ChatComponent } from './components/chat/chat.component';
import { VoiceGridComponent } from './components/voice-grid/voice-grid.component';
import { PlayerComponent } from './components/player/player.component';
import { QueueComponent } from './components/queue/queue.component';

@Component({
  selector: 'app-room',
  standalone: true,
  imports: [
    CommonModule,
    ChatComponent,
    VoiceGridComponent,
    PlayerComponent,
    QueueComponent
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

  public roomId = '';
  public isMicActive = false;
  public isMuted = false;
  public isScreenSharing = false;
  public isCameraActive = false;
  public isVoiceBusy = false;
  public isScreenShareBusy = false;
  public isCameraBusy = false;
  public isLeavingRoom = false;
  public hasScreenshare = false;
  public hasCameraActive = false;
  public memberCount = 0;
  public voiceParticipantCount = 0;

  private subs: Subscription[] = [];
  private presenceTimer: any = null;
  private membersPollTimer: any = null;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('room_id');
    if (!id) {
      this.toast.error('Phong khong hop le.');
      this.router.navigate(['/dashboard']);
      return;
    }

    this.roomId = id;
    this.loadRoomDetails();
  }

  ngOnDestroy(): void {
    this.subs.forEach(sub => sub.unsubscribe());
    this.disconnectAll();
  }

  private loadRoomDetails(): void {
    this.api.room.get(this.roomId).subscribe({
      next: (room) => {
        this.state.activeRoom$.next(room);
        this.loadRoomMembers();
        this.connectWebSockets();

        // Auto-join the voice channel on entry to enable all huddle features immediately
        setTimeout(() => {
          this.onVoiceChannelClick();
        }, 800);
      },
      error: (err) => {
        this.toast.error('Khong the tai thong tin phong: ' + err.message);
        this.router.navigate(['/dashboard']);
      }
    });
  }

  private connectWebSockets(): void {
    const token = this.state.accessToken;
    if (!token) {
      this.toast.error('Thieu access token xac thuc.');
      this.router.navigate(['/auth']);
      return;
    }

    this.chatWs.connect(this.roomId, token);
    this.playbackWs.connect(this.roomId, token);

    this.startPresenceHeartbeat();
    this.startMembersPolling();

    this.subs.push(
      this.voiceService.connected$.subscribe(connected => {
        this.isMicActive = connected;
        if (!connected) {
          this.isMuted = false;
          this.isScreenSharing = false;
          this.isCameraActive = false;
          this.isVoiceBusy = false;
        }
      }),
      this.voiceService.isMuted$.subscribe(muted => {
        this.isMuted = muted;
      }),
      this.voiceService.isScreenSharing$.subscribe(sharing => {
        this.isScreenSharing = sharing;
      }),
      this.voiceService.isCameraActive$.subscribe(active => {
        this.isCameraActive = active;
      }),
      this.voiceService.participants$.subscribe(list => {
        const screensharing = list.some(participant => participant.isScreenSharing);
        const cameraActive = list.some(participant => participant.isCameraOn);
        if ((screensharing && !this.hasScreenshare) || (cameraActive && !this.hasCameraActive)) {
          // Auto-focus on voice/video workspace when someone starts sharing screen or starts camera
          this.state.isVoiceFocused$.next(true);
        }
        this.hasScreenshare = screensharing;
        this.hasCameraActive = cameraActive;
        this.voiceParticipantCount = list.length;
      }),
      this.state.activeRoomMembers$.subscribe(members => {
        this.memberCount = members.length;
      })
    );
  }

  public async onVoiceChannelClick(): Promise<void> {
    if (this.isVoiceBusy) return;

    if (this.isMicActive) {
      this.voiceService.disconnect();
      this.toast.success('Da roi kenh voice.');
      return;
    }

    this.isVoiceBusy = true;
    this.toast.info('Dang yeu cau ket noi kenh voice...');
    this.api.voice.getToken(this.roomId).subscribe({
      next: async (res) => {
        try {
          await this.voiceService.connect(res.livekit_url, res.token);
          this.toast.success('Da tham gia kenh voice.');
        } catch (err: any) {
          this.toast.error('Loi voice call: ' + err.message);
        } finally {
          this.isVoiceBusy = false;
        }
      },
      error: (err) => {
        this.isVoiceBusy = false;
        this.toast.error('Khong the xin token voice call: ' + err.message);
      }
    });
  }

  public async onMuteToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isVoiceBusy) return;

    try {
      const targetMute = !this.isMuted;
      await this.voiceService.setMute(targetMute);
      this.toast.info(targetMute ? 'Da tat microphone.' : 'Da bat microphone.');
    } catch (err: any) {
      this.toast.error('Loi microphone: ' + err.message);
    }
  }

  public async onScreenShareToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isScreenShareBusy) return;

    try {
      this.isScreenShareBusy = true;
      const targetState = !this.isScreenSharing;
      await this.voiceService.setScreenShare(targetState);
      this.toast.success(targetState ? 'Da chia se man hinh.' : 'Da dung chia se.');
    } catch (err: any) {
      this.toast.error('Loi chia se man hinh: ' + err.message);
    } finally {
      this.isScreenShareBusy = false;
    }
  }

  public async onCameraToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isCameraBusy) return;

    try {
      this.isCameraBusy = true;
      const targetState = !this.isCameraActive;
      await this.voiceService.setCamera(targetState);
      this.toast.info(targetState ? 'Da bat camera.' : 'Da tat camera.');
    } catch (err: any) {
      this.toast.error('Loi camera: ' + err.message);
    } finally {
      this.isCameraBusy = false;
    }
  }

  public async onBackToLobbyClick(): Promise<void> {
    if (this.isLeavingRoom) return;

    this.isLeavingRoom = true;
    try {
      await this.api.room.leave(this.roomId).toPromise();
    } catch (err: any) {
      console.warn('Leave room failed:', err);
      this.toast.info('Khong dong bo duoc trang thai roi phong tren server, nhung da thoat giao dien phong.');
    } finally {
      this.disconnectAll();
      this.router.navigate(['/dashboard']);
      this.isLeavingRoom = false;
    }
  }

  public get roomModeLabel(): string {
    if (this.isScreenSharing) return 'Screen live';
    if (this.isCameraActive) return 'Camera live';
    if (this.isMicActive) return 'Voice live';
    return 'Standby';
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
}
