import { Component, OnInit, OnDestroy, inject, ElementRef, ViewChild, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ChatWsService, ChatMessage } from '../../../../core/services/websocket/chat-ws.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { ApiService } from '../../../../core/services/api.service';
import { Subscription } from 'rxjs';
import { RoomUiStateService } from '../../room-ui-state.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { MarkdownPipe } from '../../../../shared/pipes/markdown.pipe';

@Component({
  selector: 'app-room-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, MarkdownPipe],
  templateUrl: './chat.component.html',
  styleUrl: './chat.component.css'
})
export class ChatComponent implements OnInit, OnDestroy {
  private chatWs = inject(ChatWsService);
  public state = inject(StateService);
  private toast = inject(ToastService);
  private api = inject(ApiService);
  private uiState = inject(RoomUiStateService);
  private engine = inject(PlayerEngineService);

  @Input() isOpen = false;

  private profileCache = new Map<string, { username: string, display_name: string, avatar_url: string }>();

  @ViewChild('messagesContainer') private messagesContainer!: ElementRef;

  public messages: ChatMessage[] = [];
  public messageContent = '';
  public pinnedMessageContent = '';
  public isPinnedBannerVisible = false;
  public replyingTo: ChatMessage | null = null;
  public activeReactMsgId: string | null = null;

  private subs: Subscription[] = [];

  constructor() {}

  ngOnInit(): void {
    // 1. Subscribe message received
    this.subs.push(
      this.chatWs.messageReceived$.subscribe((msg: any) => {
        this.enrichMessage(msg);

        // Handle optimistic UI matching
        if (msg.client_id) {
          const idx = this.messages.findIndex(m => m.id === msg.client_id);
          if (idx !== -1) {
            this.messages[idx].id = msg.id;
            this.messages[idx].status = 'sent';
            this.messages[idx].created_at = msg.created_at;
            this.messages[idx].reply_to = msg.reply_to_id;
            setTimeout(() => this.scrollToBottom(), 50);
            return;
          }
        }

        this.messages.push(msg);
        if (!this.uiState.uiState.isChatOpen) {
          this.uiState.markUnread(1);
        }
        setTimeout(() => this.scrollToBottom(), 50);
      })
    );

    // 1b. Subscribe message error
    this.subs.push(
      this.chatWs.messageError$.subscribe((err) => {
        // Find any 'sending' message and mark as error
        const idx = this.messages.findIndex(m => m.status === 'sending');
        if (idx !== -1) {
          this.messages[idx].status = 'error';
        }
        this.toast.error('Lỗi gửi tin: ' + (err.message || 'Unknown'));
      })
    );

    this.subs.push(
      this.chatWs.translationReceived$.subscribe((translation) => {
        if (translation.kind !== 'chat') return;
        const message = this.messages.find((item) => item.id === translation.event_id);
        if (message) message.translation = translation.text;
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

    // 4. Subscribe live reactions
    this.subs.push(
      this.chatWs.liveReaction$.subscribe((reactMsg: any) => {
        if (reactMsg.message_id && reactMsg.action) {
          const m = this.messages.find(msg => msg.id === reactMsg.message_id);
          if (m) {
            if (!m.reactions) m.reactions = [];
            const existing = m.reactions.find(r => r.emoji === reactMsg.emoji);
            if (reactMsg.action === 'add') {
              if (existing) {
                if (!existing.users.includes(reactMsg.user_id)) {
                  existing.users.push(reactMsg.user_id);
                }
              } else {
                m.reactions.push({ emoji: reactMsg.emoji, users: [reactMsg.user_id] });
              }
            } else if (reactMsg.action === 'remove') {
              if (existing) {
                existing.users = existing.users.filter((u: string) => u !== reactMsg.user_id);
                if (existing.users.length === 0) {
                  m.reactions = m.reactions.filter(r => r.emoji !== reactMsg.emoji);
                }
              }
            }
          }
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

    const clientId = 'temp_' + Math.random().toString(36).substring(2, 11);
    const currentUser = this.state.user$.value;

    this.messages.push({
      id: clientId,
      sender: {
        id: currentUser?.id || 'unknown',
        username: currentUser?.username || 'unknown',
        display_name: currentUser?.display_name || 'Tôi',
        avatar_url: currentUser?.avatar_url || ''
      },
      content: content,
      reply_to: this.replyingTo ? this.replyingTo.id : null,
      created_at: new Date().toISOString(),
      status: 'sending'
    });
    setTimeout(() => this.scrollToBottom(), 50);

    this.chatWs.sendMessage(content, this.replyingTo ? this.replyingTo.id : null, clientId);
    this.messageContent = '';
    this.replyingTo = null;
  }

  public cancelReply(): void {
    this.replyingTo = null;
  }

  public getReplyMessage(replyToId: string): ChatMessage | undefined {
    return this.messages.find(m => m.id === replyToId);
  }

  public scrollToMessage(msgId: string): void {
    const el = document.getElementById('msg-' + msgId);
    if (el && this.messagesContainer) {
      this.messagesContainer.nativeElement.scrollTo({
        top: el.offsetTop - this.messagesContainer.nativeElement.offsetTop - 10,
        behavior: 'smooth'
      });
      el.classList.add('highlight-flash');
      setTimeout(() => el.classList.remove('highlight-flash'), 2000);
    }
  }

  public toggleReactMenu(msgId: string): void {
    this.activeReactMsgId = this.activeReactMsgId === msgId ? null : msgId;
  }

  public reactToMessage(msgId: string, emoji: string): void {
    // Check if user already reacted
    const msg = this.messages.find(m => m.id === msgId);
    let action: 'add' | 'remove' = 'add';
    const userId = this.state.user$.value?.id;
    if (msg && msg.reactions && userId) {
      const existing = msg.reactions.find(r => r.emoji === emoji);
      if (existing && existing.users.includes(userId)) {
        action = 'remove';
      }
    }
    
    this.chatWs.sendReaction(msgId, emoji, action);
    this.activeReactMsgId = null;
  }

  public deleteMessage(msgId: string): void {
    if (confirm('Bạn có chắc muốn thu hồi tin nhắn này?')) {
      this.chatWs.deleteMessage(msgId);
    }
  }

  public pinMessage(msgId: string): void {
    this.chatWs.pinMessage(msgId);
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

    this.api.user.getProfile(senderId, this.state.activeRoom$.value?.id).subscribe({
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

  public close(): void {
    this.uiState.toggleChat(false);
  }

  public getMessageSegments(content: string): { type: 'text' | 'link'; value: string; timeMs?: number }[] {
    const segments: { type: 'text' | 'link'; value: string; timeMs?: number }[] = [];
    const regex = /\[(\d{2}):(\d{2})\]/g;
    let lastIndex = 0;
    let match;

    while ((match = regex.exec(content)) !== null) {
      const index = match.index;
      if (index > lastIndex) {
        segments.push({
          type: 'text',
          value: content.substring(lastIndex, index)
        });
      }

      const min = parseInt(match[1], 10);
      const sec = parseInt(match[2], 10);
      const timeMs = (min * 60 + sec) * 1000;

      segments.push({
        type: 'link',
        value: match[0],
        timeMs: timeMs
      });

      lastIndex = regex.lastIndex;
    }

    if (lastIndex < content.length) {
      segments.push({
        type: 'text',
        value: content.substring(lastIndex)
      });
    }

    return segments.length > 0 ? segments : [{ type: 'text', value: content }];
  }

  public onTimestampClick(timeMs: number): void {
    this.engine.seekToMs(timeMs);
  }
}
