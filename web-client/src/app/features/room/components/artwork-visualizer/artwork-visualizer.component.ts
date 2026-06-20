import { Component, Input, OnChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Track } from '../../../../core/services/websocket/playback-ws.service';

@Component({
  selector: 'app-room-artwork-visualizer',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="art-stage" [class.playing]="isPlaying">
      <div class="halo" aria-hidden="true"></div>
      <div class="bars" aria-hidden="true">
        @for (h of barHeights; track $index) {
          <span class="bar" [style.height.%]="h" [style.animation-delay.ms]="$index * 60"></span>
        }
      </div>
      <div class="artwork">
        @if (track?.thumbnail_url) {
          <img [src]="track!.thumbnail_url" [attr.alt]="track?.title">
        } @else {
          <span class="placeholder">🎜</span>
        }
      </div>
    </div>
  `,
  styles: [`
    .art-stage { position: relative; width: 280px; height: 280px; display: flex; align-items: center; justify-content: center; }
    .artwork {
      width: 180px; height: 180px; border-radius: var(--radius-xl); overflow: hidden;
      box-shadow: var(--shadow-lg); background: var(--bg-elevated);
      display: flex; align-items: center; justify-content: center; position: relative; z-index: 2;
    }
    .artwork img { width: 100%; height: 100%; object-fit: cover; }
    .placeholder { font-size: 48px; color: var(--text-muted); }
    .halo {
      position: absolute; inset: 0; border-radius: 50%;
      background: radial-gradient(circle, var(--accent-glow-color) 0%, transparent 60%);
      opacity: 0; transition: var(--transition-slow); z-index: 1;
    }
    .art-stage.playing .halo { opacity: 1; animation: halo-pulse 2.4s var(--ease-out-expo) infinite; }
    .bars { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; gap: 4px; z-index: 1; }
    .bar { width: 5px; height: 20%; border-radius: var(--radius-full); background: var(--accent-gradient); opacity: 0.55; }
    .art-stage.playing .bar { animation: bar-bounce 0.9s var(--ease-out-expo) infinite alternate; }
    @keyframes halo-pulse { 0%,100% { transform: scale(1); opacity: 0.6; } 50% { transform: scale(1.08); opacity: 1; } }
    @keyframes bar-bounce { from { transform: scaleY(0.6); } to { transform: scaleY(1.6); } }
    @media (prefers-reduced-motion: reduce) {
      .art-stage.playing .halo { animation: none; opacity: 0.6; }
      .art-stage.playing .bar { animation: none; }
    }
  `]
})
export class ArtworkVisualizerComponent implements OnChanges {
  @Input() track: Track | null = null;
  @Input() isPlaying = false;

  public barHeights = this.seedBars();

  ngOnChanges(): void {
    // Re-seed on track/play change so the visualizer "restarts".
    this.barHeights = this.seedBars();
  }

  private seedBars(): number[] {
    return Array.from({ length: 24 }, () => 20 + Math.round(Math.random() * 70));
  }
}
