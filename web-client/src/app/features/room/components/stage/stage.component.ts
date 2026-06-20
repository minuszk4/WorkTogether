import { Component, inject, Input, ViewChild, ElementRef, AfterViewInit, OnDestroy, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { RoomUiStateService, StageMode } from '../../room-ui-state.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { VoiceService } from '../../../../core/services/voice.service';
import { StateService } from '../../../../core/services/state.service';
import { ArtworkVisualizerComponent } from '../artwork-visualizer/artwork-visualizer.component';
import { NowPlayingInfoComponent } from '../now-playing-info/now-playing-info.component';

@Component({
  selector: 'app-room-video-track',
  standalone: true,
  imports: [CommonModule],
  template: `<video #videoEl autoplay playsinline [muted]="isLocal" [style.object-fit]="objectFit"></video>`,
  styles: [`
    :host { display: block; width: 100%; height: 100%; position: relative; }
    video { width: 100%; height: 100%; border-radius: var(--radius-md); background: #000; }
  `]
})
export class RoomVideoTrackComponent implements AfterViewInit, OnDestroy, OnChanges {
  @Input() track: any;
  @Input() isLocal: boolean = false;
  @Input() objectFit: 'cover' | 'contain' = 'cover';
  @ViewChild('videoEl') videoEl!: ElementRef<HTMLVideoElement>;

  ngAfterViewInit() {
    this.attachTrack();
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['track']) {
      const prevTrack = changes['track'].previousValue;
      const currentTrack = changes['track'].currentValue;
      console.log('[RoomVideoTrackComponent Debug] ngOnChanges track. Changed:', !!currentTrack);

      if (prevTrack && this.videoEl) {
        try {
          prevTrack.detach(this.videoEl.nativeElement);
        } catch (e) {
          console.warn('Failed to detach previous track:', e);
        }
      }

      this.attachTrack();
    }
  }

  ngOnDestroy() {
    this.detachTrack();
  }

  private attachTrack() {
    if (this.track && this.videoEl) {
      console.log('[RoomVideoTrackComponent Debug] Attaching track:', this.track.sid || this.track.name || 'unnamed');
      try {
        this.track.attach(this.videoEl.nativeElement);
      } catch (e) {
        console.error('[RoomVideoTrackComponent Debug] Failed to attach track:', e);
      }
    } else {
      console.log('[RoomVideoTrackComponent Debug] Cannot attach track. Has track:', !!this.track, 'Has videoEl:', !!this.videoEl);
    }
  }

  private detachTrack() {
    if (this.track && this.videoEl) {
      try {
        this.track.detach(this.videoEl.nativeElement);
      } catch (e) {
        console.warn('Failed to detach track:', e);
      }
    }
  }
}

@Component({
  selector: 'app-room-stage',
  standalone: true,
  imports: [CommonModule, ArtworkVisualizerComponent, NowPlayingInfoComponent, RoomVideoTrackComponent],
  template: `
    <div class="stage">
      @switch (stageMode) {
        @case ('screenshare') {
          <div class="screenshare-layout">
            <div class="stage-media" [class.screenshare-grid]="screenshareParticipants.length > 1" [class.screenshare-main]="screenshareParticipants.length <= 1">
              @for (p of screenshareParticipants; track p.sid) {
                <div class="screenshare-grid-item">
                  <app-room-video-track [track]="getScreenshareTrack(p)" [isLocal]="p.isLocal" [objectFit]="'contain'"></app-room-video-track>
                  <div class="video-label">{{ p.display_name }} đang chia sẻ màn hình</div>
                </div>
              }
            </div>
            
            @if (cameraParticipants.length > 0) {
              <div class="screenshare-camera-strip">
                @for (p of cameraParticipants; track p.sid) {
                  <div class="strip-item">
                    <app-room-video-track [track]="getCameraTrack(p)" [isLocal]="p.isLocal"></app-room-video-track>
                    <div class="strip-label">{{ p.display_name }}</div>
                  </div>
                }
              </div>
            }
          </div>
        }
        @case ('video') {
          <div class="stage-media video-grid">
            @for (p of cameraParticipants; track p.sid) {
              <div class="video-grid-item">
                <app-room-video-track [track]="getCameraTrack(p)" [isLocal]="p.isLocal"></app-room-video-track>
                <div class="video-label">{{ p.display_name }}</div>
              </div>
            }
          </div>
        }
        @default {
          <div class="stage-music">
            <app-room-artwork-visualizer [track]="engine.currentTrack" [isPlaying]="isPlaying"></app-room-artwork-visualizer>
            <div class="stage-meta">
              <app-room-now-playing [track]="engine.currentTrack"></app-room-now-playing>
            </div>
          </div>
        }
      }
    </div>
  `,
  styleUrl: './stage.component.css'
})
export class StageComponent implements OnDestroy {
  public engine = inject(PlayerEngineService);
  private uiState = inject(RoomUiStateService);
  public voiceService = inject(VoiceService);
  private state = inject(StateService);

  public stageMode: StageMode = 'music-only';
  public isPlaying = false;
  private subs: Subscription[] = [];

  constructor() {
    this.subs.push(
      this.uiState.changes$.subscribe(s => (this.stageMode = s.stageMode)),
      this.engine.currentState$.subscribe(s => (this.isPlaying = s === 'playing'))
    );
  }

  get screenshareParticipants() {
    const list = this.voiceService.participants$.value || [];
    const found = list.filter(p => p.isScreenSharing);
    console.log('[StageComponent Debug] screenshareParticipants count:', found.length);
    const members = this.state.activeRoomMembers$.value || [];
    return found.map(p => {
      const member = members.find(m => m.user_id === p.identity || m.id === p.identity);
      return {
        ...p,
        display_name: member?.display_name || p.identity || 'Unknown'
      };
    });
  }

  getScreenshareTrack(p: any) {
    if (!p || !p.raw) return null;
    const publications = Array.from((p.raw as any).trackPublications.values()) as any[];
    const pub = publications.find(pub => pub.source === 'screen_share' || (pub.track && pub.track.source === 'screen_share'));
    if (!pub) return null;
    return pub.track || (pub as any).videoTrack;
  }

  get cameraParticipants() {
    const list = this.voiceService.participants$.value || [];
    const found = list.filter(p => p.isCameraOn);
    console.log('[StageComponent Debug] cameraParticipants count:', found.length);
    const members = this.state.activeRoomMembers$.value || [];
    return found.map(p => {
      const member = members.find(m => m.user_id === p.identity || m.id === p.identity);
      return {
        ...p,
        display_name: member?.display_name || p.identity || 'Unknown'
      };
    });
  }

  getCameraTrack(p: any) {
    if (!p || !p.raw) return null;
    const publications = Array.from((p.raw as any).trackPublications.values()) as any[];
    const pub = publications.find(pub => pub.source === 'camera' || (pub.track && pub.track.source === 'camera'));
    if (!pub) return null;
    return pub.track || (pub as any).videoTrack;
  }

  ngOnDestroy() {
    this.subs.forEach(s => s.unsubscribe());
  }
}

