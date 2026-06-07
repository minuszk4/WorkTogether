import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

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
  public connected$ = new Subject<boolean>();

  constructor() {}

  public connect(roomId: string, token: string): void {
    this.roomId = roomId;
    const wsUrl = `ws://localhost:8080/api/v1/rooms/${roomId}/chat/ws?token=${token}`;

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
        case 'chat:message_deleted':
          this.messageDeleted$.next(msg.payload.id);
          break;
        case 'chat:pin_updated':
          this.pinnedUpdate$.next(msg.payload);
          break;
        default:
          console.log('[Chat WS] Sự kiện chưa xử lý:', msg.event);
      }
    } catch (e) {
      console.error('Lỗi phân tích WS message:', e);
    }
  }

  public sendMessage(content: string, replyToId: string | null = null): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error('Không thể gửi tin nhắn. Kênh chat chưa sẵn sàng.');
      return;
    }

    const payload = {
      event: 'chat:send_message',
      room_id: this.roomId,
      payload: {
        content: content,
        reply_to_id: replyToId
      }
    };

    this.socket.send(JSON.stringify(payload));
  }

  public disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.roomId = null;
  }
}
