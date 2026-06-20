import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

export interface PillParticipant {
  sid: string;
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
    <div class="voice-pill" [class.speaking]="p.isSpeaking" [class.muted]="p.isMuted">
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
    </div>
  `,
  styles: [`
    .voice-pill { display: flex; align-items: center; gap: 8px; padding: 6px 12px; border-radius: var(--radius-full); background: var(--bg-glass); backdrop-filter: blur(12px); border: 1px solid var(--border-color); transition: var(--transition-normal); }
    .voice-pill.speaking { border-color: var(--accent-primary); box-shadow: var(--accent-glow); transform: scale(1.04); }
    .vp-avatar { position: relative; width: 28px; height: 28px; border-radius: 50%; overflow: hidden; background: var(--bg-elevated); display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: var(--text-primary); }
    .vp-avatar img { width: 100%; height: 100%; object-fit: cover; }
    .vp-ring { position: absolute; inset: -3px; border: 2px solid var(--accent-primary); border-radius: 50%; animation: ring-pulse 1.2s var(--ease-out-expo) infinite; }
    @keyframes ring-pulse { 0%,100% { opacity: 0.7; } 50% { opacity: 1; } }
    .vp-name { font-size: var(--text-sm-size); color: var(--text-primary); white-space: nowrap; display: flex; align-items: center; }
    .vp-mic { width: 14px; height: 14px; color: var(--danger); }
    .vp-playback-status { position: absolute; inset: 0; background: rgba(0, 0, 0, 0.45); display: flex; align-items: center; justify-content: center; opacity: 0.75; transition: opacity 0.2s; }
    .status-icon { width: 10px; height: 10px; color: #fff; }
    .vp-unsynced-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; background-color: var(--warning, #f59e0b); margin-left: 6px; vertical-align: middle; }
    @media (prefers-reduced-motion: reduce) { .vp-ring { animation: none; } }
  `]
})
export class VoicePillComponent {
  @Input({ required: true }) p!: PillParticipant;
  get initials(): string {
    return (this.p?.display_name || 'WT').slice(0, 2).toUpperCase();
  }
}
