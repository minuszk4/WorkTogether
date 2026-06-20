import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';

@Component({
  selector: 'app-room-transport',
  standalone: true,
  imports: [CommonModule, IconButtonComponent],
  template: `
    <div class="transport">
      <app-icon-button label="Phát trước" (clicked)="prev.emit()">
        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 6h2v12H6zM9.5 12l8.5 6V6z"/></svg>
      </app-icon-button>

      <button class="play-main" type="button" [attr.aria-label]="isPlaying ? 'Tạm dừng' : 'Phát'" (click)="playPause.emit()">
        @if (isPlaying) {
          <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
        } @else {
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>
        }
      </button>

      <app-icon-button label="Phát tiếp" (clicked)="next.emit()">
        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M16 6h2v12h-2zM6 18l8.5-6L6 6z"/></svg>
      </app-icon-button>
    </div>
  `,
  styles: [`
    .transport { display: flex; align-items: center; gap: 14px; }
    .play-main {
      width: 46px; height: 46px; border: none; border-radius: 50%;
      background: var(--accent-gradient); color: #05121f; cursor: pointer;
      display: flex; align-items: center; justify-content: center;
      box-shadow: var(--shadow-accent); transition: var(--transition-fast);
    }
    .play-main:hover { transform: scale(1.06); }
    .play-main:active { transform: scale(0.96); }
    .play-main svg { width: 20px; height: 20px; }
  `]
})
export class TransportControlsComponent {
  @Input() isPlaying = false;
  @Output() playPause = new EventEmitter<void>();
  @Output() prev = new EventEmitter<void>();
  @Output() next = new EventEmitter<void>();
}
