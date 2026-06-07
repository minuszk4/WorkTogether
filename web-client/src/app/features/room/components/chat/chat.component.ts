import { Component, OnInit, OnDestroy, inject, ElementRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ChatWsService, ChatMessage } from '../../../../core/services/websocket/chat-ws.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { ApiService } from '../../../../core/services/api.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-room-chat',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './chat.component.html',
  styleUrl: './chat.component.css'
})
export class ChatComponent implements OnInit, OnDestroy {
  private chatWs = inject(ChatWsService);
  public state = inject(StateService);
  private toast = inject(ToastService);
  private api = inject(ApiService);

  private profileCache = new Map<string, { username: string, display_name: string, avatar_url: string }>();

  @ViewChild('messagesContainer') private messagesContainer!: ElementRef;

  public messages: ChatMessage[] = [];
  public messageContent = '';
  public pinnedMessageContent = '';
  public isPinnedBannerVisible = false;

  private subs: Subscription[] = [];

  constructor() {}

  ngOnInit(): void {
    // 1. Subscribe message received
    this.subs.push(
      this.chatWs.messageReceived$.subscribe((msg) => {
        this.enrichMessage(msg);
        this.messages.push(msg);
        setTimeout(() => this.scrollToBottom(), 50);
      })
    );

    // 2. Subscribe message deleted
    this.subs.push(
      this.chatWs.messageDeleted$.subscribe((msgId) => {
        const msg = this.messages.find(m => m.id === msgId);
        if (msg) {
          msg.content = 'Tin nhắn đã bị thu hồi.';
        }
      })
    );

    // 3. Subscribe pinned message update
    this.subs.push(
      this.chatWs.pinnedUpdate$.subscribe((pinned) => {
        if (pinned && pinned.content) {
          this.pinnedMessageContent = pinned.content;
          this.isPinnedBannerVisible = true;
        } else {
          this.pinnedMessageContent = '';
          this.isPinnedBannerVisible = false;
        }
      })
    );

    // Initial placeholder message
    this.messages.push({
      id: 'system-connected',
      sender: { id: 'system', username: 'system', display_name: 'Hệ thống', avatar_url: '' },
      content: 'Đã kết nối vào phòng chat.',
      reply_to: null,
      created_at: new Date().toISOString()
    });
    setTimeout(() => this.scrollToBottom(), 50);
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }



  public onSendSubmit(event: Event): void {
    event.preventDefault();
    const content = this.messageContent.trim();
    if (!content) return;

    this.chatWs.sendMessage(content);
    this.messageContent = '';
  }

  private scrollToBottom(): void {
    try {
      if (this.messagesContainer) {
        const element = this.messagesContainer.nativeElement;
        element.scrollTop = element.scrollHeight;
      }
    } catch (err) {
      console.warn('Scroll to bottom failed:', err);
    }
  }

  public formatTime(timeStr: string): string {
    try {
      const timeVal = new Date(timeStr);
      return `${String(timeVal.getHours()).padStart(2, '0')}:${String(timeVal.getMinutes()).padStart(2, '0')}`;
    } catch {
      return '00:00';
    }
  }

  private enrichMessage(msg: any): void {
    const senderId = msg.sender_id || (msg.sender ? msg.sender.id : null);
    if (!senderId) {
      if (!msg.sender) {
        msg.sender = { id: 'unknown', username: 'unknown', display_name: 'Unknown', avatar_url: '' };
      }
      return;
    }

    const cached = this.profileCache.get(senderId);
    if (cached) {
      msg.sender = {
        id: senderId,
        username: cached.username,
        display_name: cached.display_name,
        avatar_url: cached.avatar_url
      };
      return;
    }

    msg.sender = {
      id: senderId,
      username: senderId === 'system' ? 'system' : 'user_' + senderId.substring(0, 8),
      display_name: senderId === 'system' ? 'Hệ thống' : 'Thành viên ' + senderId.substring(0, 8),
      avatar_url: ''
    };

    if (senderId === 'system') {
      return;
    }

    this.api.user.getProfile(senderId).subscribe({
      next: (res) => {
        if (res) {
          const display_name = res.display_name || res.username || 'Thành viên ' + senderId.substring(0, 8);
          const username = res.username || 'user_' + senderId.substring(0, 8);
          const avatar_url = res.avatar_url || '';

          this.profileCache.set(senderId, { username, display_name, avatar_url });

          msg.sender.username = username;
          msg.sender.display_name = display_name;
          msg.sender.avatar_url = avatar_url;

          this.messages.forEach(m => {
            const mSenderId = (m as any).sender_id || (m.sender ? m.sender.id : null);
            if (mSenderId === senderId && m.sender) {
              m.sender.username = username;
              m.sender.display_name = display_name;
              m.sender.avatar_url = avatar_url;
            }
          });
        }
      },
      error: () => {}
    });
  }
}
