import { Injectable } from '@angular/core';
import { Subject, BehaviorSubject } from 'rxjs';
import { environment } from '../../../../environments/environment';

export interface ChatMessage {
  id: string;
  sender?: {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
  };
  content: string;
  reply_to: string | null;
  created_at: string;
  status?: 'sending' | 'sent' | 'error';
  reactions?: { emoji: string; users: string[] }[];
}

export interface RoomModeChange {
  mode: 'chill' | 'focus' | 'collaborate';
  changed_by: string;
}

export interface RoomEvent {
  event: string;
  payload: any;
}

@Injectable({
  providedIn: 'root'
})
export class ChatWsService {
  private socket: WebSocket | null = null;
  private roomId: string | null = null;

  public messageReceived$ = new Subject<ChatMessage>();
  public messageDeleted$ = new Subject<string>();
  public pinnedUpdate$ = new Subject<any>();
  public messageError$ = new Subject<any>();
  public connected$ = new Subject<boolean>();

  public listenerStates$ = new BehaviorSubject<Record<string, { is_playing: boolean; position_ms: number; updated_at: number }>>({});
  public liveReaction$ = new Subject<{ user_id: string; emoji: string }>();
  public roomVibe$ = new BehaviorSubject<{ current_vibe: string; vibe_scores: Record<string, number> } | null>(null);
  public roomMode$ = new Subject<RoomModeChange>();
  public roomEvent$ = new Subject<RoomEvent>();

  constructor() {}

  public connect(roomId: string, token: string): void {
    this.roomId = roomId;
    const wsUrl = `${environment.wsUrl}/api/v1/rooms/${roomId}/chat/ws?token=${token}`;

    this.socket = new WebSocket(wsUrl);

    this.socket.onopen = () => {
      console.log('[Chat WS] Kết nối thành công.');
      this.connected$.next(true);
    };

    this.socket.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.socket.onerror = (err) => {
      console.error('[Chat WS] Lỗi kết nối:', err);
      this.connected$.next(false);
    };

    this.socket.onclose = () => {
      console.log('[Chat WS] Đã ngắt kết nối.');
      this.connected$.next(false);
    };
  }

  private handleMessage(dataStr: string): void {
    try {
      const msg = JSON.parse(dataStr);
      switch (msg.event) {
        case 'chat:message_received':
          this.messageReceived$.next(msg.payload);
          break;
        case 'chat:message_edited':
          // Currently not tracked directly here, but could emit
          break;
        case 'chat:message_deleted':
          this.messageDeleted$.next(msg.payload.message_id || msg.payload.id);
          break;
        case 'chat:pin_updated':
        case 'chat:message_pinned':
        case 'chat:message_unpinned':
          this.pinnedUpdate$.next(msg.payload);
          break;
        case 'chat:reaction_updated':
          // Re-emit liveReaction$ or handle specially.
          // In Spec 2, we should emit this to liveReaction$ or another subject.
          // Let's emit it to liveReaction$ to reuse existing subject.
          this.liveReaction$.next({ user_id: msg.payload.user_id, emoji: msg.payload.emoji, ...msg.payload });
          break;
        case 'chat:error':
          this.messageError$.next(msg.payload);
          break;
        case 'presence:listener_states':
          this.listenerStates$.next(msg.payload.user_states);
          break;
        case 'presence:reaction_broadcast':
          this.liveReaction$.next({ user_id: msg.payload.user_id, emoji: msg.payload.emoji });
          break;
        case 'presence:vibe_tick':
          this.roomVibe$.next(msg.payload);
          break;
        case 'room:mode_changed':
          this.roomMode$.next(msg.payload);
          break;
        default:
          if (typeof msg.event === 'string' && msg.event.startsWith('session:')) {
            this.roomEvent$.next({ event: msg.event, payload: msg.payload });
            break;
          }
          console.log('[Chat WS] Sự kiện chưa xử lý:', msg.event);
      }
    } catch (e) {
      console.error('Lỗi phân tích WS message:', e);
    }
  }

  public sendMessage(content: string, replyToId: string | null = null, clientId: string | null = null): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error('Không thể gửi tin nhắn. Kênh chat chưa sẵn sàng.');
      return;
    }

    const payload = {
      event: 'chat:send_message',
      room_id: this.roomId,
      payload: {
        client_id: clientId,
        content: content,
        reply_to_id: replyToId
      }
    };

    this.socket.send(JSON.stringify(payload));
  }

  public updatePresenceState(isPlaying: boolean, positionMs: number): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'presence:state_change',
      room_id: this.roomId,
      payload: { is_playing: isPlaying, position_ms: positionMs }
    }));
  }

  public deleteMessage(messageId: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'chat:delete_message',
      room_id: this.roomId,
      payload: { message_id: messageId }
    }));
  }

  public pinMessage(messageId: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'chat:pin_message',
      room_id: this.roomId,
      payload: { message_id: messageId }
    }));
  }

  public sendReaction(messageId: string, emoji: string, action: 'add' | 'remove'): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'chat:react',
      room_id: this.roomId,
      payload: { message_id: messageId, emoji: emoji, action: action }
    }));
  }

  public sendLiveReaction(emoji: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'presence:send_reaction',
      room_id: this.roomId,
      payload: { emoji }
    }));
  }

  public disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.roomId = null;
  }
}
