import { Injectable, NgZone, OnDestroy, inject } from '@angular/core';
import { BehaviorSubject, Subscription } from 'rxjs';
import { PlaybackWsService, PlaybackState, Track } from '../../../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../../../core/services/voice.service';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

/**
 * Pure playback-sync engine extracted from the legacy PlayerComponent.
 * Hosts the YouTube IFrame / SoundCloud Widget / HTML5 audio players by element id.
 * UI components read observables here and call command methods.
 */
@Injectable({ providedIn: 'root' })
export class PlayerEngineService implements OnDestroy {
  private playbackWs = inject(PlaybackWsService);
  private voiceService = inject(VoiceService);
  private zone = inject(NgZone);
  private chatWs = inject(ChatWsService);
  private heartbeatInterval: any = null;

  private ytPlayer: any = null;
  private isReady = false;
  private isSyncingLocal = false;
  private progressTimer: any = null;
  private scWidget: any = null;
  private isScReady = false;
  private audioEl: HTMLAudioElement | null = null;

  private readonly _currentState$ = new BehaviorSubject<'playing' | 'paused' | 'stopped'>('stopped');
  private readonly _currentTrack$ = new BehaviorSubject<Track | null>(null);
  private readonly _progressMs$ = new BehaviorSubject<number>(0);
  private readonly _progressPct$ = new BehaviorSubject<number>(0);
  private readonly _volume$ = new BehaviorSubject<number>(80);
  private readonly _hasScreenshare$ = new BehaviorSubject<boolean>(false);

  private lastServerPosition = 0;
  private lastUpdatedAt = 0;
  private subs: Subscription[] = [];

  constructor() {
    this.subs.push(
      this.playbackWs.playbackSync$.subscribe(state => {
        if (state) this.handlePlaybackSync(state);
      }),
      this.voiceService.participants$.subscribe(list => {
        this._hasScreenshare$.next((list || []).some(p => p.isScreenSharing));
      })
    );
  }

  /* ── Public observables / getters ───────────────────────── */
  public get currentState() { return this._currentState$.value; }
  public get currentTrack() { return this._currentTrack$.value; }
  public get volume() { return this._volume$.value; }
  public get currentState$() { return this._currentState$.asObservable(); }
  public get currentTrack$() { return this._currentTrack$.asObservable(); }
  public get progressMs$() { return this._progressMs$.asObservable(); }
  public get progressPct$() { return this._progressPct$.asObservable(); }
  public get volume$() { return this._volume$.asObservable(); }
  public get hasScreenshare$() { return this._hasScreenshare$.asObservable(); }

  /* ── Bootstrap (called by host component AfterViewInit) ─── */
  public loadApis(): void {
    if (!(window as any)['YT'] || !(window as any)['YT'].Player) {
      const tag = document.createElement('script');
      tag.src = 'https://www.youtube.com/iframe_api';
      document.getElementsByTagName('script')[0].parentNode?.insertBefore(tag, document.getElementsByTagName('script')[0]);
    }
    if (!(window as any)['SC']) {
      const tag = document.createElement('script');
      tag.src = 'https://w.soundcloud.com/player/api.js';
      document.head.appendChild(tag);
    }
  }

  public initPlayers(): void {
    this.initYoutubePlayer();
    this.initSoundCloudPlayer();
    this.initHtml5Audio();
  }

  /* ── Engine internals (ported from PlayerComponent) ─────── */
  private safeCall(method: string, args: any[] = [], defaultValue: any = null): any {
    if (this.ytPlayer && typeof this.ytPlayer[method] === 'function') {
      try { return this.ytPlayer[method](...args); } catch (e) { console.warn(`YT ${method}:`, e); }
    }
    return defaultValue;
  }

  private initYoutubePlayer(): void {
    const setupPlayer = () => {
      this.ytPlayer = new (window as any).YT.Player('youtube-player-element', {
        height: '100%', width: '100%', videoId: '',
        playerVars: { autoplay: 0, controls: 0, disablekb: 1, fs: 0, rel: 0 },
        events: {
          onReady: (event: any) => {
            this.isReady = true; this.ytPlayer = event.target;
            this.safeCall('setVolume', [this._volume$.value]);
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
      const existing = (window as any).onYouTubeIframeAPIReady;
      (window as any).onYouTubeIframeAPIReady = () => { if (existing) existing(); setupPlayer(); };
      const interval = setInterval(() => {
        if ((window as any).YT && (window as any).YT.Player) { clearInterval(interval); setupPlayer(); }
      }, 500);
    }
  }

  private initSoundCloudPlayer(): void {
    const iframe = document.getElementById('soundcloud-player-element') as HTMLIFrameElement;
    if (!iframe) return;
    if ((window as any).SC && (window as any).SC.Widget) {
      this.scWidget = (window as any).SC.Widget(iframe);
      this.scWidget.bind((window as any).SC.Widget.Events.READY, () => {
        this.isScReady = true; this.scWidget.setVolume(this._volume$.value);
        if (this.currentTrack && this.currentTrack.source === 'soundcloud') this.syncSoundCloudPlayer();
      });
      this.scWidget.bind((window as any).SC.Widget.Events.PLAY, () => this.handleScStateChange('playing'));
      this.scWidget.bind((window as any).SC.Widget.Events.PAUSE, () => this.handleScStateChange('paused'));
    } else {
      setTimeout(() => this.initSoundCloudPlayer(), 500);
    }
  }

  private initHtml5Audio(): void {
    this.audioEl = document.getElementById('html5-audio-element') as HTMLAudioElement;
    if (this.audioEl) {
      this.audioEl.volume = this._volume$.value / 100;
      this.audioEl.onplay = () => this.handleHtml5StateChange('playing');
      this.audioEl.onpause = () => this.handleHtml5StateChange('paused');
    }
  }

  private handlePlaybackSync(state: PlaybackState): void {
    const isPlaying = state.state === 'playing';
    
    // Send state change instantly on play/pause/seek events
    this.chatWs.updatePresenceState(isPlaying, state.position_ms);

    this._currentState$.next(state.state);
    this._currentTrack$.next(state.current_track);
    this.lastServerPosition = state.position_ms;
    this.lastUpdatedAt = state.updated_at;
    this.syncPlayerWithServerState();
    this.startProgressTimer();

    if (isPlaying) {
      this.startHeartbeat();
    } else {
      this.stopHeartbeat();
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      if (this.currentState === 'playing') {
        this.chatWs.updatePresenceState(true, this.getLocalProgress());
      }
    }, 5000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private syncPlayerWithServerState(): void {
    const track = this.currentTrack;
    if (!track) { this.stopAllPlayers(); return; }
    const source = track.source || 'youtube';
    if (source === 'youtube') { this.pauseSoundCloud(); this.pauseHtml5Audio(); this.syncYoutubePlayer(); }
    else if (source === 'soundcloud') { this.stopYoutube(); this.pauseHtml5Audio(); this.syncSoundCloudPlayer(); }
    else { this.stopYoutube(); this.pauseSoundCloud(); this.syncHtml5Audio(); }
  }

  private stopAllPlayers(): void { this.stopYoutube(); this.pauseSoundCloud(); this.pauseHtml5Audio(); }
  private stopYoutube(): void { if (this.ytPlayer && this.isReady) this.safeCall('stopVideo'); }
  private pauseSoundCloud(): void { if (this.scWidget && this.isScReady) this.scWidget.pause(); }
  private pauseHtml5Audio(): void { this.audioEl?.pause(); }

  private syncYoutubePlayer(): void {
    if (!this.ytPlayer || !this.isReady) return;
    const track = this.currentTrack!;
    const videoId = this.extractYoutubeId(track.source_url);
    if (!videoId) return;
    let currentYTVideo = '';
    try { const vd = this.safeCall('getVideoData'); if (vd) currentYTVideo = vd.video_id; } catch (e) {}

    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    const targetPosSec = targetPosMS / 1000;
    this.isSyncingLocal = true;

    if (currentYTVideo !== videoId) {
      this.safeCall('cueVideoById', [{ videoId, startSeconds: targetPosSec > 0 ? targetPosSec : 0 }]);
    }
    setTimeout(() => {
      try {
        const playerState = this.safeCall('getPlayerState', [], -1);
        const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
        if (this.currentState === 'playing') {
          const localTimeSec = this.safeCall('getCurrentTime', [], 0);
          if (Math.abs(localTimeSec - targetPosSec) > 1.5) this.safeCall('seekTo', [targetPosSec, true]);
          if (playerState !== ytPlaying) this.safeCall('playVideo');
        } else if (this.currentState === 'paused') {
          this.safeCall('seekTo', [targetPosSec, true]); this.safeCall('pauseVideo');
        } else { this.safeCall('stopVideo'); }
      } catch (e) { console.warn('YT adjust:', e); }
      setTimeout(() => { this.isSyncingLocal = false; }, 500);
    }, currentYTVideo !== videoId ? 800 : 50);
  }

  private syncSoundCloudPlayer(): void {
    if (!this.scWidget || !this.isScReady || !this.currentTrack) return;
    const track = this.currentTrack;
    const currentURL = track.source_url;
    this.scWidget.getCurrentSound((sound: any) => {
      const widgetURL = sound ? sound.permalink_url : '';
      const serverTimeEst = this.playbackWs.getEstimatedServerTime();
      let targetPosMS = this.lastServerPosition;
      if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
      this.isSyncingLocal = true;
      const performSync = () => {
        this.scWidget.seekTo(targetPosMS);
        if (this.currentState === 'playing') this.scWidget.play();
        else if (this.currentState === 'paused') this.scWidget.pause();
        setTimeout(() => { this.isSyncingLocal = false; }, 500);
      };
      if (!widgetURL || !this.isUrlsSame(widgetURL, currentURL)) {
        this.scWidget.load(currentURL, { auto_play: this.currentState === 'playing', callback: () => performSync() });
      } else { performSync(); }
    });
  }

  private syncHtml5Audio(): void {
    if (!this.audioEl || !this.currentTrack) return;
    const track = this.currentTrack;
    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    const targetPosSec = targetPosMS / 1000;
    this.isSyncingLocal = true;
    if (this.audioEl.src !== track.source_url) { this.audioEl.src = track.source_url; this.audioEl.load(); }
    const drift = Math.abs(this.audioEl.currentTime - targetPosSec);
    if (drift > 1.5) this.audioEl.currentTime = targetPosSec;
    if (this.currentState === 'playing') this.audioEl.play().catch(err => console.warn('HTML5 play:', err));
    else this.audioEl.pause();
    setTimeout(() => { this.isSyncingLocal = false; }, 500);
  }

  private handleScStateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack) return;
    this.scWidget.getPosition((posMS: number) => {
      let action: 'play' | 'pause' | '' = '';
      if (state === 'playing' && this.currentState !== 'playing') action = 'play';
      else if (state === 'paused' && this.currentState === 'playing') action = 'pause';
      if (action) this.playbackWs.sendControlCommand(action, Math.floor(posMS));
    });
  }

  private handleHtml5StateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack || !this.audioEl) return;
    const posMS = Math.floor(this.audioEl.currentTime * 1000);
    let action: 'play' | 'pause' | '' = '';
    if (state === 'playing' && this.currentState !== 'playing') action = 'play';
    else if (state === 'paused' && this.currentState === 'playing') action = 'pause';
    if (action) this.playbackWs.sendControlCommand(action, posMS);
  }

  private handlePlayerStateChange(state: number): void {
    if (this.isSyncingLocal || !this.currentTrack) return;
    let action: 'play' | 'pause' | '' = '';
    let currentPosMS = 0;
    try {
      currentPosMS = Math.floor((this.safeCall('getCurrentTime', [], 0) || 0) * 1000);
      const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
      const ytPaused = (window as any).YT?.PlayerState?.PAUSED ?? 2;
      if (state === ytPlaying && this.currentState !== 'playing') action = 'play';
      else if (state === ytPaused && this.currentState === 'playing') action = 'pause';
    } catch (e) {}
    if (action) this.playbackWs.sendControlCommand(action, currentPosMS);
  }

  private isUrlsSame(u1: string, u2: string): boolean {
    const clean = (u: string) => u.replace('https://', '').replace('http://', '').replace('www.', '').split('?')[0];
    return clean(u1) === clean(u2);
  }

  private startProgressTimer(): void {
    if (this.progressTimer) clearInterval(this.progressTimer);
    this.progressTimer = setInterval(() => {
      const track = this.currentTrack;
      if (!track) { this._progressMs$.next(0); this._progressPct$.next(0); return; }
      const ms = this.getLocalProgress();
      this._progressMs$.next(ms);
      this._progressPct$.next((ms / track.duration_ms) * 100);
    }, 200);
  }

  /* ── Public commands (used by UI components) ────────────── */
  public getLocalProgress(): number {
    if (this.currentState !== 'playing' || this.lastUpdatedAt === 0) return this.lastServerPosition;
    const elapsed = this.playbackWs.getEstimatedServerTime() - this.lastUpdatedAt;
    let pos = this.lastServerPosition + elapsed;
    if (this.currentTrack && pos > this.currentTrack.duration_ms) pos = this.currentTrack.duration_ms;
    return pos;
  }

  public togglePlayPause(): void {
    if (!this.currentTrack) return;
    const action = this.currentState === 'playing' ? 'pause' : 'play';
    this.playbackWs.sendControlCommand(action, this.getLocalProgress());
  }

  public seekMsFromFraction(fraction: number): number {
    return Math.max(0, Math.min(1, fraction)) * (this.currentTrack?.duration_ms ?? 0);
  }

  public seekToMs(ms: number): void {
    if (!this.currentTrack) return;
    this.playbackWs.sendControlCommand('seek', ms);
  }

  public seekFromFraction(fraction: number): void {
    this.seekToMs(this.seekMsFromFraction(fraction));
  }

  public setVolumeFromFraction(fraction: number): void {
    const v = Math.round(Math.max(0, Math.min(1, fraction)) * 100);
    this._volume$.next(v);
    const source = this.currentTrack?.source || 'youtube';
    if (source === 'youtube') this.safeCall('setVolume', [v]);
    else if (source === 'soundcloud' && this.scWidget && this.isScReady) this.scWidget.setVolume(v);
    else if (this.audioEl) this.audioEl.volume = v / 100;
  }

  public formatTime(ms: number): string {
    const secTotal = Math.floor(ms / 1000);
    return `${String(Math.floor(secTotal / 60)).padStart(2, '0')}:${String(secTotal % 60).padStart(2, '0')}`;
  }

  private extractYoutubeId(urlStr: string): string | null {
    const reg = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
    const m = urlStr.match(reg);
    return (m && m[2].length === 11) ? m[2] : null;
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    if (this.progressTimer) clearInterval(this.progressTimer);
    this.stopHeartbeat();
    if (this.ytPlayer && this.ytPlayer.destroy) this.ytPlayer.destroy();
    if (this.audioEl) { this.audioEl.pause(); this.audioEl.src = ''; }
  }
}
