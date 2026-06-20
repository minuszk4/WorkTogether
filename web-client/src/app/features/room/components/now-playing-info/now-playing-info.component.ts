import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Track } from '../../../../core/services/websocket/playback-ws.service';

@Component({
  selector: 'app-room-now-playing',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="now-playing" [class.compact]="compact">
      @if (track?.thumbnail_url) {
        <img class="np-thumb" [src]="track!.thumbnail_url" [attr.alt]="track?.title">
      } @else {
        <div class="np-thumb np-thumb-ph" aria-hidden="true">🎜</div>
      }
      <div class="np-meta">
        <span class="np-title" [title]="track ? track!.title : ''">{{ track ? track!.title : 'Chưa phát bài hát nào' }}</span>
        <span class="np-artist">{{ track ? track!.artist : 'Hàng đợi đang trống' }}</span>
      </div>
    </div>
  `,
  styles: [`
    .now-playing { display: flex; align-items: center; gap: 10px; min-width: 0; }
    .np-thumb { width: 44px; height: 44px; border-radius: var(--radius-md); object-fit: cover; background: var(--bg-elevated); flex-shrink: 0; }
    .np-thumb-ph { display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 20px; }
    .np-meta { display: flex; flex-direction: column; min-width: 0; }
    .np-title { font-weight: 600; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-size: var(--text-sm-size); }
    .np-artist { font-size: var(--text-xs-size); color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  `]
})
export class NowPlayingInfoComponent {
  @Input() track: Track | null = null;
  @Input() compact = false;
}
