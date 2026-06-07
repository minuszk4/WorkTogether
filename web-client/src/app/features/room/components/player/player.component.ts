import { Component, OnInit, OnDestroy, inject, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { PlaybackWsService, PlaybackState, Track } from '../../../../core/services/websocket/playback-ws.service';
import { StateService } from '../../../../core/services/state.service';
import { VoiceService } from '../../../../core/services/voice.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-room-player',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './player.component.html',
  styleUrl: './player.component.css'
})
export class PlayerComponent implements OnInit, OnDestroy, AfterViewInit {
  public playbackWs = inject(PlaybackWsService);
  public state = inject(StateService);
  public voiceService = inject(VoiceService);

  private ytPlayer: any = null;
  private isReady = false;
  private isSyncingLocal = false;
  private progressTimer: any = null;

  public hasScreenshare = false;

  // SoundCloud state
  private scWidget: any = null;
  private isScReady = false;

  // HTML5 audio state
  private audioEl: HTMLAudioElement | null = null;

  public currentState: 'playing' | 'paused' | 'stopped' = 'stopped';
  public currentTrack: Track | null = null;
  public lastServerPosition = 0;
  public lastUpdatedAt = 0;

  // UI state
  public displayProgress = 0; // ms
  public displayProgressPct = 0;
  public volume = 80;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    // Load YouTube IFrame API if not already present
    if (!(window as any)['YT'] || !(window as any)['YT'].Player) {
      const tag = document.createElement('script');
      tag.src = 'https://www.youtube.com/iframe_api';
      const firstScriptTag = document.getElementsByTagName('script')[0];
      firstScriptTag.parentNode?.insertBefore(tag, firstScriptTag);
    }

    // Load SoundCloud Widget API if not already present
    if (!(window as any)['SC']) {
      const tag = document.createElement('script');
      tag.src = 'https://w.soundcloud.com/player/api.js';
      document.head.appendChild(tag);
    }

    this.subs.push(
      this.playbackWs.playbackSync$.subscribe(state => {
        if (state) {
          this.handlePlaybackSync(state);
        }
      }),
      this.voiceService.participants$.subscribe(list => {
        this.hasScreenshare = (list || []).some(p => p.isScreenSharing);
      })
    );
  }

  ngAfterViewInit(): void {
    this.initYoutubePlayer();
    this.initSoundCloudPlayer();
    this.initHtml5Audio();
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    if (this.progressTimer) {
      clearInterval(this.progressTimer);
    }
    if (this.ytPlayer && this.ytPlayer.destroy) {
      this.ytPlayer.destroy();
    }
    if (this.audioEl) {
      this.audioEl.pause();
      this.audioEl.src = '';
    }
  }

  private safeCall(method: string, args: any[] = [], defaultValue: any = null): any {
    if (this.ytPlayer && typeof this.ytPlayer[method] === 'function') {
      try {
        return this.ytPlayer[method](...args);
      } catch (e) {
        console.warn(`Error calling YT player method ${method}:`, e);
      }
    }
    return defaultValue;
  }

  private initYoutubePlayer(): void {
    const setupPlayer = () => {
      this.ytPlayer = new (window as any).YT.Player('youtube-player-element', {
        height: '100%',
        width: '100%',
        videoId: '',
        playerVars: {
          autoplay: 0,
          controls: 0,
          disablekb: 1,
          fs: 0,
          rel: 0
        },
        events: {
          onReady: (event: any) => {
            this.isReady = true;
            this.ytPlayer = event.target;
            this.safeCall('setVolume', [this.volume]);
            this.syncPlayerWithServerState();
          },
          onStateChange: (event: any) => {
            this.ytPlayer = event.target;
            this.handlePlayerStateChange(event.data);
          }
        }
      });
    };

    if ((window as any).YT && (window as any).YT.Player) {
      setupPlayer();
    } else {
      // If window.onYouTubeIframeAPIReady is called
      const existingCallback = (window as any).onYouTubeIframeAPIReady;
      (window as any).onYouTubeIframeAPIReady = () => {
        if (existingCallback) existingCallback();
        setupPlayer();
      };
      
      // Fallback polling if the global hook is missed
      const interval = setInterval(() => {
        if ((window as any).YT && (window as any).YT.Player) {
          clearInterval(interval);
          setupPlayer();
        }
      }, 500);
    }
  }

  private handlePlaybackSync(state: PlaybackState): void {
    this.currentState = state.state;
    this.currentTrack = state.current_track;
    this.lastServerPosition = state.position_ms;
    this.lastUpdatedAt = state.updated_at;

    this.syncPlayerWithServerState();
    this.startProgressTimer();
  }

  private initSoundCloudPlayer(): void {
    const iframe = document.getElementById('soundcloud-player-element') as HTMLIFrameElement;
    if (!iframe) return;

    if ((window as any).SC && (window as any).SC.Widget) {
      this.scWidget = (window as any).SC.Widget(iframe);
      this.scWidget.bind((window as any).SC.Widget.Events.READY, () => {
        this.isScReady = true;
        this.scWidget.setVolume(this.volume);
        if (this.currentTrack && this.currentTrack.source === 'soundcloud') {
          this.syncSoundCloudPlayer();
        }
      });
      this.scWidget.bind((window as any).SC.Widget.Events.PLAY, () => {
        this.handleScStateChange('playing');
      });
      this.scWidget.bind((window as any).SC.Widget.Events.PAUSE, () => {
        this.handleScStateChange('paused');
      });
    } else {
      setTimeout(() => this.initSoundCloudPlayer(), 500);
    }
  }

  private initHtml5Audio(): void {
    this.audioEl = document.getElementById('html5-audio-element') as HTMLAudioElement;
    if (this.audioEl) {
      this.audioEl.volume = this.volume / 100;
      this.audioEl.onplay = () => this.handleHtml5StateChange('playing');
      this.audioEl.onpause = () => this.handleHtml5StateChange('paused');
    }
  }

  private syncPlayerWithServerState(): void {
    if (!this.currentTrack) {
      this.stopAllPlayers();
      return;
    }

    const source = this.currentTrack.source || 'youtube';

    if (source === 'youtube') {
      this.pauseSoundCloud();
      this.pauseHtml5Audio();
      this.syncYoutubePlayer();
    } else if (source === 'soundcloud') {
      this.stopYoutube();
      this.pauseHtml5Audio();
      this.syncSoundCloudPlayer();
    } else {
      // generic or upload
      this.stopYoutube();
      this.pauseSoundCloud();
      this.syncHtml5Audio();
    }
  }

  private stopAllPlayers(): void {
    this.stopYoutube();
    this.pauseSoundCloud();
    this.pauseHtml5Audio();
  }

  private stopYoutube(): void {
    if (this.ytPlayer && this.isReady) {
      this.safeCall('stopVideo');
    }
  }

  private pauseSoundCloud(): void {
    if (this.scWidget && this.isScReady) {
      this.scWidget.pause();
    }
  }

  private pauseHtml5Audio(): void {
    if (this.audioEl) {
      this.audioEl.pause();
    }
  }

  private syncYoutubePlayer(): void {
    if (!this.ytPlayer || !this.isReady) return;

    const videoId = this.extractYoutubeId(this.currentTrack!.source_url);
    if (!videoId) return;

    let currentYTVideo = '';
    try {
      const videoData = this.safeCall('getVideoData');
      if (videoData) {
        currentYTVideo = videoData.video_id;
      }
    } catch (e) {}

    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') {
      targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    }
    const targetPosSec = targetPosMS / 1000;

    this.isSyncingLocal = true;

    if (currentYTVideo !== videoId) {
      this.safeCall('cueVideoById', [{
        videoId: videoId,
        startSeconds: targetPosSec > 0 ? targetPosSec : 0
      }]);
    }

    setTimeout(() => {
      try {
        const playerState = this.safeCall('getPlayerState', [], -1);
        const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
        const ytPaused = (window as any).YT?.PlayerState?.PAUSED ?? 2;
        
        if (this.currentState === 'playing') {
          const localTimeSec = this.safeCall('getCurrentTime', [], 0);
          const drift = Math.abs(localTimeSec - targetPosSec);

          if (drift > 1.5) {
            this.safeCall('seekTo', [targetPosSec, true]);
          }
          
          if (playerState !== ytPlaying) {
            this.safeCall('playVideo');
          }
        } else if (this.currentState === 'paused') {
          this.safeCall('seekTo', [targetPosSec, true]);
          this.safeCall('pauseVideo');
        } else {
          this.safeCall('stopVideo');
        }
      } catch (e) {
        console.warn('Error adjusting YT video state:', e);
      }

      setTimeout(() => {
        this.isSyncingLocal = false;
      }, 500);

    }, currentYTVideo !== videoId ? 800 : 50);
  }

  private syncSoundCloudPlayer(): void {
    if (!this.scWidget || !this.isScReady || !this.currentTrack) return;

    const currentURL = this.currentTrack.source_url;
    this.scWidget.getCurrentSound((sound: any) => {
      const widgetTrackURL = sound ? sound.permalink_url : '';
      
      const serverTimeEst = this.playbackWs.getEstimatedServerTime();
      let targetPosMS = this.lastServerPosition;
      if (this.currentState === 'playing') {
        targetPosMS += (serverTimeEst - this.lastUpdatedAt);
      }

      this.isSyncingLocal = true;

      const performSync = () => {
        this.scWidget.seekTo(targetPosMS);
        if (this.currentState === 'playing') {
          this.scWidget.play();
        } else if (this.currentState === 'paused') {
          this.scWidget.pause();
        }
        setTimeout(() => { this.isSyncingLocal = false; }, 500);
      };

      if (!widgetTrackURL || !this.isUrlsSame(widgetTrackURL, currentURL)) {
        this.scWidget.load(currentURL, {
          auto_play: this.currentState === 'playing',
          callback: () => {
            performSync();
          }
        });
      } else {
        performSync();
      }
    });
  }

  private syncHtml5Audio(): void {
    if (!this.audioEl || !this.currentTrack) return;

    const currentURL = this.currentTrack.source_url;
    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') {
      targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    }
    const targetPosSec = targetPosMS / 1000;

    this.isSyncingLocal = true;

    if (this.audioEl.src !== currentURL) {
      this.audioEl.src = currentURL;
      this.audioEl.load();
    }

    const localTimeSec = this.audioEl.currentTime;
    const drift = Math.abs(localTimeSec - targetPosSec);
    if (drift > 1.5) {
      this.audioEl.currentTime = targetPosSec;
    }

    if (this.currentState === 'playing') {
      this.audioEl.play().catch(err => console.warn('HTML5 play failed:', err));
    } else {
      this.audioEl.pause();
    }

    setTimeout(() => {
      this.isSyncingLocal = false;
    }, 500);
  }

  private handleScStateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack) return;

    this.scWidget.getPosition((posMS: number) => {
      let action: 'play' | 'pause' | '' = '';
      if (state === 'playing' && this.currentState !== 'playing') {
        action = 'play';
      } else if (state === 'paused' && this.currentState === 'playing') {
        action = 'pause';
      }

      if (action !== '') {
        this.playbackWs.sendControlCommand(action, Math.floor(posMS));
      }
    });
  }

  private handleHtml5StateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack || !this.audioEl) return;

    const posMS = Math.floor(this.audioEl.currentTime * 1000);
    let action: 'play' | 'pause' | '' = '';
    if (state === 'playing' && this.currentState !== 'playing') {
      action = 'play';
    } else if (state === 'paused' && this.currentState === 'playing') {
      action = 'pause';
    }

    if (action !== '') {
      this.playbackWs.sendControlCommand(action, posMS);
    }
  }

  private isUrlsSame(url1: string, url2: string): boolean {
    const clean = (u: string) => u.replace('https://', '').replace('http://', '').replace('www.', '').split('?')[0];
    return clean(url1) === clean(url2);
  }

  private handlePlayerStateChange(state: number): void {
    if (this.isSyncingLocal || !this.currentTrack) return;

    let action: 'play' | 'pause' | '' = '';
    let currentPosMS = 0;
    try {
      currentPosMS = Math.floor((this.safeCall('getCurrentTime', [], 0) || 0) * 1000);
      const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
      const ytPaused = (window as any).YT?.PlayerState?.PAUSED ?? 2;

      if (state === ytPlaying && this.currentState !== 'playing') {
        action = 'play';
      } else if (state === ytPaused && this.currentState === 'playing') {
        action = 'pause';
      }
    } catch (e) {}

    if (action !== '') {
      this.playbackWs.sendControlCommand(action, currentPosMS);
    }
  }

  public onPlayPauseClick(): void {
    if (!this.currentTrack) return;
    const action = this.currentState === 'playing' ? 'pause' : 'play';
    const currentPos = this.getLocalProgress();
    this.playbackWs.sendControlCommand(action, currentPos);
  }

  public toggleFocus(): void {
    this.state.isVoiceFocused$.next(!this.state.isVoiceFocused$.value);
  }

  public onScrubberClick(event: MouseEvent): void {
    if (!this.currentTrack) return;
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    const pct = (event.clientX - rect.left) / rect.width;
    const seekMS = Math.floor(pct * this.currentTrack.duration_ms);
    this.playbackWs.sendControlCommand('seek', seekMS);
  }

  public onVolumeClick(event: MouseEvent): void {
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    const pct = (event.clientX - rect.left) / rect.width;
    this.volume = Math.floor(pct * 100);
    
    if (this.currentTrack) {
      const source = this.currentTrack.source || 'youtube';
      if (source === 'youtube') {
        this.safeCall('setVolume', [this.volume]);
      } else if (source === 'soundcloud' && this.scWidget && this.isScReady) {
        this.scWidget.setVolume(this.volume);
      } else if (this.audioEl) {
        this.audioEl.volume = this.volume / 100;
      }
    } else {
      this.safeCall('setVolume', [this.volume]);
      if (this.scWidget && this.isScReady) {
        this.scWidget.setVolume(this.volume);
      }
      if (this.audioEl) {
        this.audioEl.volume = this.volume / 100;
      }
    }
  }

  public getLocalProgress(): number {
    if (this.currentState !== 'playing' || this.lastUpdatedAt === 0) {
      return this.lastServerPosition;
    }
    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    const elapsed = serverTimeEst - this.lastUpdatedAt;
    let pos = this.lastServerPosition + elapsed;
    
    if (this.currentTrack && pos > this.currentTrack.duration_ms) {
      pos = this.currentTrack.duration_ms;
    }
    return pos;
  }

  private startProgressTimer(): void {
    if (this.progressTimer) {
      clearInterval(this.progressTimer);
    }

    this.progressTimer = setInterval(() => {
      if (!this.currentTrack) {
        this.displayProgress = 0;
        this.displayProgressPct = 0;
        return;
      }

      this.displayProgress = this.getLocalProgress();
      this.displayProgressPct = (this.displayProgress / this.currentTrack.duration_ms) * 100;
    }, 200);
  }

  public formatTime(ms: number): string {
    const secTotal = Math.floor(ms / 1000);
    const min = Math.floor(secTotal / 60);
    const sec = secTotal % 60;
    return `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
  }

  private extractYoutubeId(urlStr: string): string | null {
    const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
    const match = urlStr.match(regExp);
    return (match && match[2].length === 11) ? match[2] : null;
  }

  public togglePlayerFullscreen(): void {
    const container = document.querySelector('.player-viewport-shell');
    if (container) {
      if (container.requestFullscreen) {
        container.requestFullscreen();
      } else if ((container as any).webkitRequestFullscreen) { /* Safari */
        (container as any).webkitRequestFullscreen();
      } else if ((container as any).msRequestFullscreen) { /* IE11 */
        (container as any).msRequestFullscreen();
      }
    }
  }
}
