import { Component, Input, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { ApiService } from '../../../../core/services/api.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';

export interface PillParticipant {
  sid: string;
  identity?: string;
  display_name: string;
  avatar_url?: string;
  isSpeaking: boolean;
  isMuted: boolean;
  isCurrentUser: boolean;
  isPlaying?: boolean;
  isUnsynced?: boolean;
}

@Component({
  selector: 'app-room-voice-pill',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="voice-pill" [class.speaking]="p.isSpeaking" [class.muted]="p.isMuted" [class.react-bounce]="isReacting" (click)="toggleMenu($event)">
      <div class="vp-avatar">
        @if (p.avatar_url) {
          <img [src]="p.avatar_url" [alt]="p.display_name">
        } @else {
          <span>{{ initials }}</span>
        }
        @if (p.isSpeaking) { <span class="vp-ring" aria-hidden="true"></span> }
        
        @if (p.isPlaying !== undefined) {
          <div class="vp-playback-status" [class.playing]="p.isPlaying" [attr.aria-label]="p.isPlaying ? 'Đang nghe' : 'Tạm dừng'">
            @if (p.isPlaying) {
              <svg viewBox="0 0 24 24" fill="currentColor" class="status-icon"><path d="M8 5v14l11-7z"/></svg>
            } @else {
              <svg viewBox="0 0 24 24" fill="currentColor" class="status-icon"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
            }
          </div>
        }

        <div class="vp-floaters-container" aria-hidden="true">
          @for (f of floaters; track f.id) {
            <span class="vp-floater">{{ f.emoji }}</span>
          }
        </div>
      </div>
      <span class="vp-name">
        {{ p.display_name }}{{ p.isCurrentUser ? ' (Bạn)' : '' }}
        @if (p.isUnsynced) {
          <span class="vp-unsynced-dot" title="Lệch pha" aria-label="Lệch pha"></span>
        }
      </span>
      @if (p.isMuted) {
        <svg class="vp-mic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Đã tắt mic"><line x1="1" y1="1" x2="23" y2="23"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"/></svg>
      }

      @if (menuOpen) {
        <div class="vp-menu" (click)="$event.stopPropagation()">
          @if (isHost && !p.isCurrentUser) {
            <button class="vp-menu-item" (click)="assignDj()">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"></path><circle cx="6" cy="18" r="3"></circle><circle cx="18" cy="16" r="3"></circle></svg>
              Gán quyền Guest DJ
            </button>
            <button class="vp-menu-item danger" (click)="revokeDj()">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"></path></svg>
              Thu hồi Guest DJ
            </button>
          } @else {
            <div class="vp-menu-item disabled">Không có hành động</div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .voice-pill { display: flex; align-items: center; gap: 8px; padding: 6px 12px; border-radius: var(--radius-full); background: var(--bg-glass); backdrop-filter: blur(12px); border: 1px solid var(--border-color); transition: var(--transition-normal); }
    .voice-pill.speaking { border-color: var(--accent-primary); box-shadow: var(--accent-glow); transform: scale(1.04); }
    .voice-pill.react-bounce { animation: pill-bounce 1s cubic-bezier(0.175, 0.885, 0.32, 1.275); }
    @keyframes pill-bounce { 0%, 100% { transform: scale(1); } 50% { transform: scale(1.15); } }
    .vp-avatar { position: relative; width: 28px; height: 28px; border-radius: 50%; overflow: hidden; background: var(--bg-elevated); display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: var(--text-primary); }
    .vp-avatar img { width: 100%; height: 100%; object-fit: cover; }
    .vp-ring { position: absolute; inset: -3px; border: 2px solid var(--accent-primary); border-radius: 50%; animation: ring-pulse 1.2s var(--ease-out-expo) infinite; }
    @keyframes ring-pulse { 0%,100% { opacity: 0.7; } 50% { opacity: 1; } }
    .vp-name { font-size: var(--text-sm-size); color: var(--text-primary); white-space: nowrap; display: flex; align-items: center; }
    .vp-mic { width: 14px; height: 14px; color: var(--danger); }
    .vp-playback-status { position: absolute; inset: 0; background: rgba(0, 0, 0, 0.45); display: flex; align-items: center; justify-content: center; opacity: 0.75; transition: opacity 0.2s; }
    .status-icon { width: 10px; height: 10px; color: #fff; }
    .vp-unsynced-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; background-color: var(--warning, #f59e0b); margin-left: 6px; vertical-align: middle; }
    .vp-floaters-container { position: absolute; inset: 0; pointer-events: none; overflow: visible; }
    .vp-floater { position: absolute; left: 50%; top: 50%; font-size: 16px; transform: translate(-50%, -50%); animation: float-up 1.5s ease-out forwards; }
    @keyframes float-up { 0% { transform: translate(-50%, -50%) scale(0.5); opacity: 0; } 20% { opacity: 1; } 100% { transform: translate(-50%, -60px) scale(1.2); opacity: 0; } }
    .vp-menu { position: absolute; top: 100%; left: 0; margin-top: 8px; background: var(--bg-elevated); border: 1px solid var(--border-color); border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.5); padding: 4px; z-index: 50; display: flex; flex-direction: column; min-width: 180px; }
    .vp-menu-item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border: none; background: transparent; color: var(--text-primary); cursor: pointer; border-radius: 4px; text-align: left; font-size: 13px; }
    .vp-menu-item:hover:not(.disabled) { background: var(--bg-hover); }
    .vp-menu-item.danger { color: var(--danger); }
    .vp-menu-item.disabled { opacity: 0.5; cursor: default; }
    .vp-menu-item svg { width: 14px; height: 14px; }
    @media (prefers-reduced-motion: reduce) {
      .vp-ring { animation: none; }
      .voice-pill.react-bounce { animation: none; }
      .vp-floater { animation: float-up-reduced 1.5s ease-out forwards; }
      @keyframes float-up-reduced { 0% { opacity: 0; } 20% { opacity: 1; } 100% { opacity: 0; } }
    }
  `]
})
export class VoicePillComponent implements OnInit, OnDestroy {
  @Input({ required: true }) p!: PillParticipant;
  
  private chatWs = inject(ChatWsService);
  private api = inject(ApiService);
  private state = inject(StateService);
  private toast = inject(ToastService);
  private sub: Subscription | null = null;
  private clickOutSub: () => void = () => {};

  public isReacting = false;
  public menuOpen = false;
  public floaters: { id: number; emoji: string }[] = [];
  private floaterId = 0;

  get initials(): string {
    return (this.p?.display_name || 'WT').slice(0, 2).toUpperCase();
  }

  get isHost(): boolean {
    return this.state.roomMemberRole$.value === 'OWNER';
  }

  ngOnInit(): void {
    this.sub = this.chatWs.liveReaction$.subscribe(rx => {
      if (rx && rx.user_id === this.p.identity) {
        this.triggerReaction(rx.emoji);
      }
    });
    this.clickOutSub = () => { if (this.menuOpen) this.menuOpen = false; };
    window.addEventListener('click', this.clickOutSub);
  }

  ngOnDestroy(): void {
    if (this.sub) this.sub.unsubscribe();
    window.removeEventListener('click', this.clickOutSub);
  }

  public toggleMenu(event: MouseEvent): void {
    event.stopPropagation();
    if (this.isHost && !this.p.isCurrentUser) {
      this.menuOpen = !this.menuOpen;
    }
  }

  public assignDj(): void {
    this.menuOpen = false;
    const roomId = this.state.activeRoom$.value?.id;
    if (!roomId || !this.p.identity) return;
    
    this.api.playback.assignGuestDj(roomId, this.p.identity).subscribe({
      next: () => this.toast.success(`Đã gán quyền DJ cho ${this.p.display_name}`),
      error: (err: any) => this.toast.error('Lỗi: ' + (err.error?.error?.message || err.message))
    });
  }

  public revokeDj(): void {
    this.menuOpen = false;
    const roomId = this.state.activeRoom$.value?.id;
    if (!roomId) return;

    this.api.playback.revokeGuestDj(roomId).subscribe({
      next: () => this.toast.success(`Đã thu hồi quyền DJ của ${this.p.display_name}`),
      error: (err: any) => this.toast.error('Lỗi: ' + (err.error?.error?.message || err.message))
    });
  }

  public triggerReaction(emoji: string): void {
    this.isReacting = true;
    setTimeout(() => { this.isReacting = false; }, 1200);

    const id = this.floaterId++;
    this.floaters.push({ id, emoji });
    setTimeout(() => {
      this.floaters = this.floaters.filter(f => f.id !== id);
    }, 1500);
  }
}
