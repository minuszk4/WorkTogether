import { Injectable } from '@angular/core';
import { Subject, BehaviorSubject } from 'rxjs';
import { environment } from '../../../../environments/environment';

export interface Track {
  id: string;
  title: string;
  artist: string;
  thumbnail_url: string;
  duration_ms: number;
  source_url: string;
  source?: string;
}

export interface PlaybackState {
  state: 'playing' | 'paused' | 'stopped';
  current_track: Track | null;
  position_ms: number;
  updated_at: number;
}

@Injectable({
  providedIn: 'root'
})
export class PlaybackWsService {
  private socket: WebSocket | null = null;
  private roomId: string | null = null;
  private pingInterval: any = null;

  // NTP Sync metrics
  public clockOffset = 0; // theta
  public rtt = 0;

  public playbackSync$ = new BehaviorSubject<PlaybackState | null>(null);
  public connected$ = new BehaviorSubject<boolean>(false);
  public guestDj$ = new BehaviorSubject<{ userId: string; endsAt: number } | null>(null);
  public poll$ = new BehaviorSubject<{ candidates: any[]; duration: number } | null>(null);
  public pollVotes$ = new BehaviorSubject<Record<string, number>>({});
  public pollEnd$ = new Subject<{ winner: { track_id: string; title: string } | null }>();
  public error$ = new Subject<{ code: string; message: string }>();

  constructor() {}

  public connect(roomId: string, token: string): void {
    this.roomId = roomId;
    const wsUrl = `${environment.wsUrl}/api/v1/rooms/${roomId}/playback/ws?token=${token}`;

    this.socket = new WebSocket(wsUrl);

    this.socket.onopen = () => {
      console.log('[Playback WS] Kết nối thành công.');
      this.connected$.next(true);

      // Start NTP clock offset measurements
      this.sendNTPPing();
      this.pingInterval = setInterval(() => this.sendNTPPing(), 10000);
    };

    this.socket.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.socket.onerror = (err) => {
      console.error('[Playback WS] Lỗi kết nối:', err);
      this.connected$.next(false);
    };

    this.socket.onclose = () => {
      console.log('[Playback WS] Ngắt kết nối.');
      this.connected$.next(false);
      this.cleanup();
    };
  }

  private sendNTPPing(): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;

    const payload = {
      event: 'sync:ping',
      payload: {
        t1: Date.now()
      }
    };
    this.socket.send(JSON.stringify(payload));
  }

  private handleMessage(dataStr: string): void {
    try {
      const msg = JSON.parse(dataStr);
      switch (msg.event) {
        case 'sync:pong':
          this.calculateClockOffset(msg.payload);
          break;
        case 'playback:sync':
          this.handlePlaybackSync(msg.payload);
          break;
        case 'dj:takeover':
          this.guestDj$.next({
            userId: msg.payload?.user_id,
            endsAt: msg.payload?.ends_at
          });
          break;
        case 'dj:released':
          this.guestDj$.next(null);
          break;
        case 'poll:start':
          this.poll$.next({
            candidates: msg.payload?.candidates || [],
            duration: msg.payload?.duration || 30000
          });
          this.pollVotes$.next({});
          break;
        case 'poll:update':
          this.pollVotes$.next(msg.payload || {});
          break;
        case 'poll:end':
          this.poll$.next(null);
          this.pollVotes$.next({});
          this.pollEnd$.next({
            winner: msg.payload?.winner || null
          });
          break;
        case 'playback:error':
          this.error$.next({
            code: msg.payload?.code || 'UNKNOWN_ERROR',
            message: msg.payload?.message || ''
          });
          break;
        default:
          console.log('[Playback WS] Sự kiện chưa xử lý:', msg.event);
      }
    } catch (e) {
      console.error('Lỗi phân tích Playback WS message:', e);
    }
  }

  private calculateClockOffset(payload: any): void {
    const t4 = Date.now();
    const t1 = payload.t1;
    const t2 = payload.t2; // Server received ping
    const t3 = payload.t3; // Server sent pong

    this.rtt = (t4 - t1) - (t3 - t2);
    this.clockOffset = ((t2 - t1) + (t3 - t4)) / 2;

    console.log(`[NTP Sync] RTT: ${this.rtt}ms, Clock Offset: ${this.clockOffset}ms`);
  }

  private handlePlaybackSync(payload: any): void {
    let current_track: Track | null = null;
    if (payload.current_track_id) {
      current_track = {
        id: payload.current_track_id,
        title: payload.title || 'Unknown Title',
        artist: payload.artist || 'Unknown Artist',
        thumbnail_url: payload.thumbnail_url || '',
        duration_ms: payload.duration_ms,
        source_url: payload.source_url
      };
    }

    const state: PlaybackState = {
      state: payload.state || 'stopped',
      current_track,
      position_ms: payload.position_ms || 0,
      updated_at: payload.updated_at || Date.now()
    };

    this.playbackSync$.next(state);
  }

  public sendControlCommand(action: 'play' | 'pause' | 'seek', positionMS: number): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error('Không thể gửi lệnh điều khiển. Kênh playback chưa sẵn sàng.');
      return;
    }

    const currentState = this.playbackSync$.value;
    if (!currentState || !currentState.current_track) return;

    const payload = {
      event: 'playback:control',
      payload: {
        action: action,
        track_id: currentState.current_track.id,
        position_ms: positionMS,
        title: currentState.current_track.title,
        artist: currentState.current_track.artist,
        thumbnail_url: currentState.current_track.thumbnail_url,
        duration_ms: currentState.current_track.duration_ms,
        source_url: currentState.current_track.source_url
      }
    };

    this.socket.send(JSON.stringify(payload));
  }

  public voteForTrack(trackId: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error('Không thể vote. Kênh playback chưa sẵn sàng.');
      return;
    }
    const payload = {
      event: 'poll:vote',
      payload: {
        track_id: trackId
      }
    };
    this.socket.send(JSON.stringify(payload));
  }

  public getEstimatedServerTime(): number {
    return Date.now() + this.clockOffset;
  }

  private cleanup(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  public disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.roomId = null;
    this.guestDj$.next(null);
    this.poll$.next(null);
    this.pollVotes$.next({});
    this.cleanup();
  }
}
