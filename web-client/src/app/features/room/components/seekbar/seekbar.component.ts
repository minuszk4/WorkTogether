import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Progress/seek bar. Emits fraction [0..1] on click. */
@Component({
  selector: 'app-room-seekbar',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="seek-bar" (click)="onClick($event)" role="slider"
         [attr.aria-valuenow]="Math.round(pct)" aria-valuemin="0" aria-valuemax="100" aria-label="Seek">
      <div class="seek-track">
        <div class="seek-fill" [style.width.%]="pct"></div>
        <div class="seek-thumb" [style.left.%]="pct"></div>
      </div>
    </div>
  `,
  styles: [`
    .seek-bar { cursor: pointer; padding: 8px 0; }
    .seek-track { height: 4px; border-radius: var(--radius-full); background: var(--bg-elevated); position: relative; transition: var(--transition-fast); }
    .seek-bar:hover .seek-track { height: 6px; }
    .seek-fill { height: 100%; border-radius: inherit; background: var(--accent-gradient); }
    .seek-thumb { position: absolute; top: 50%; width: 12px; height: 12px; transform: translate(-50%, -50%); border-radius: 50%; background: var(--accent-primary); box-shadow: var(--accent-glow); opacity: 0; transition: var(--transition-fast); }
    .seek-bar:hover .seek-thumb { opacity: 1; }
  `]
})
export class SeekbarComponent {
  @Input() pct = 0;
  @Output() seek = new EventEmitter<number>();

  protected Math = Math;

  onClick(event: MouseEvent): void {
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    const fraction = (event.clientX - rect.left) / rect.width;
    this.seek.emit(fraction);
  }
}
