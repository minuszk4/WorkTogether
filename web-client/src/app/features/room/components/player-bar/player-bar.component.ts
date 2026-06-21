import { AfterViewInit, Component, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { RoomUiStateService } from '../../room-ui-state.service';
import { NowPlayingInfoComponent } from '../now-playing-info/now-playing-info.component';
import { TransportControlsComponent } from '../transport-controls/transport-controls.component';
import { SeekbarComponent } from '../seekbar/seekbar.component';
import { VolumeControlComponent } from '../volume-control/volume-control.component';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';
import { StateService } from '../../../../core/services/state.service';

@Component({
  selector: 'app-room-player-bar',
  standalone: true,
  imports: [
    CommonModule, NowPlayingInfoComponent, TransportControlsComponent,
    SeekbarComponent, VolumeControlComponent, IconButtonComponent
  ],
  templateUrl: './player-bar.component.html',
  styleUrl: './player-bar.component.css'
})
export class PlayerBarComponent implements AfterViewInit, OnDestroy {
  public engine = inject(PlayerEngineService);
  public uiState = inject(RoomUiStateService);
  private playbackWs = inject(PlaybackWsService);
  private state = inject(StateService);

  get guestDjName(): string {
    const dj = this.playbackWs.guestDj$.value;
    if (!dj) return '';
    const member = this.state.activeRoomMembers$.value.find(m => m.user_id === dj.userId);
    return member?.display_name || 'Guest DJ';
  }

  get isPlaybackLocked(): boolean {
    const dj = this.playbackWs.guestDj$.value;
    if (!dj) return false;
    const currentUserId = this.state.user?.id;
    const isOwner = this.state.roomMemberRole$.value === 'OWNER';
    return currentUserId !== dj.userId && !isOwner;
  }

  public isPlaying = false;
  public pct = 0;
  public progressMs = 0;
  public volumePct = 80;
  public queueOpen = true;
  public chatOpen = false;
  public unreadCount = 0;

  private subs: Subscription[] = [];

  ngAfterViewInit(): void {
    this.engine.loadApis();
    // Defer init so the host iframe/audio elements exist in the DOM.
    setTimeout(() => this.engine.initPlayers(), 0);

    this.subs.push(
      this.engine.currentState$.subscribe(s => (this.isPlaying = s === 'playing')),
      this.engine.progressPct$.subscribe(p => (this.pct = p)),
      this.engine.progressMs$.subscribe(ms => (this.progressMs = ms)),
      this.engine.volume$.subscribe(v => (this.volumePct = v)),
      this.uiState.changes$.subscribe(st => {
        this.queueOpen = st.isQueueOpen;
        this.chatOpen = st.isChatOpen;
        this.unreadCount = st.unreadCount;
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  onPlayPause(): void { this.engine.togglePlayPause(); }
  onSeek(fraction: number): void { this.engine.seekFromFraction(fraction); }
  onVolume(fraction: number): void { this.engine.setVolumeFromFraction(fraction); }
  onPrev(): void { /* playback API has no prev; left as no-op hook for future */ }
  onNext(): void { /* no-op hook for future */ }
  toggleQueue(): void { this.uiState.toggleQueue(); }
  toggleChat(): void { this.uiState.toggleChat(); }
}
