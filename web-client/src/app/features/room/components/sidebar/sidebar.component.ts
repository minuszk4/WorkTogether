import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';

@Component({
  selector: 'app-room-sidebar',
  standalone: true,
  imports: [CommonModule, IconButtonComponent],
  template: `
    <div class="sidebar">
      <div class="sidebar-top">
        <app-icon-button label="Quay về Lobby" (clicked)="back.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
        </app-icon-button>

        <app-icon-button label="Kênh voice" [variant]="isMicActive ? 'active' : 'default'" [disabled]="isVoiceBusy" (clicked)="voiceChannel.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 18v-6a9 9 0 0 1 18 0v6"/><path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z"/></svg>
        </app-icon-button>
		<span class="voice-status" [class]="'voice-status state-' + connectionState" [title]="connectionLabel"></span>

        <app-icon-button [label]="isMuted ? 'Bật microphone' : 'Tắt microphone'" [variant]="isMicActive && !isMuted ? 'active' : 'default'" [disabled]="!isMicActive || isVoiceBusy" (clicked)="muteToggle.emit()">
          @if (isMicActive && isMuted) {
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="1" y1="1" x2="23" y2="23"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"/><path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"/><line x1="12" y1="19" x2="12" y2="23"/></svg>
          } @else {
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>
          }
        </app-icon-button>

        <app-icon-button label="Chia sẻ màn hình" [variant]="isScreenSharing ? 'active' : 'default'" [disabled]="!isMicActive || isScreenShareBusy" (clicked)="screenShareToggle.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><path d="M8 21h8M12 17v4"/></svg>
        </app-icon-button>

        <app-icon-button label="Camera" [variant]="isCameraActive ? 'active' : 'default'" [disabled]="!isMicActive || isCameraBusy" (clicked)="cameraToggle.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M23 7l-7 5 7 5V7z"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
        </app-icon-button>
      </div>

      <div class="sidebar-bottom">
        <div class="sb-avatar" [title]="displayName">
          @if (avatarUrl) {
            <img [src]="avatarUrl" [alt]="displayName">
          } @else {
            <span>{{ initials }}</span>
          }
        </div>
      </div>
    </div>
  `,
  styleUrl: './sidebar.component.css'
})
export class SidebarComponent {
  @Input() isMicActive = false;
  @Input() isMuted = false;
  @Input() isScreenSharing = false;
  @Input() isCameraActive = false;
  @Input() isVoiceBusy = false;
  @Input() isScreenShareBusy = false;
  @Input() isCameraBusy = false;
	@Input() connectionState = 'disconnected';
  @Input() avatarUrl = '';
  @Input() displayName = '';

  @Output() back = new EventEmitter<void>();
  @Output() voiceChannel = new EventEmitter<void>();
  @Output() muteToggle = new EventEmitter<void>();
  @Output() screenShareToggle = new EventEmitter<void>();
  @Output() cameraToggle = new EventEmitter<void>();

  get initials(): string {
    return (this.displayName || 'WT').slice(0, 2).toUpperCase();
  }

	get connectionLabel(): string {
	  const labels: Record<string, string> = {
		connecting: 'Voice đang kết nối',
		connected: 'Voice đã kết nối',
		reconnecting: 'Voice đang kết nối lại',
		degraded: 'Kết nối voice yếu',
		disconnected: 'Voice chưa kết nối'
	  };
	  return labels[this.connectionState] || labels['disconnected'];
	}
}
