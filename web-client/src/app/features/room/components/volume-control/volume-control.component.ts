import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Volume slider. Emits fraction [0..1] on click. */
@Component({
  selector: 'app-room-volume-control',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="volume">
      <svg class="vol-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M11 5L6 9H2v6h4l5 4V5z"/>
        <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
      </svg>
      <div class="volume-bar" (click)="onClick($event)" role="slider"
           [attr.aria-valuenow]="volumePct" aria-valuemin="0" aria-valuemax="100" aria-label="Volume">
        <div class="volume-fill" [style.width.%]="volumePct"></div>
      </div>
    </div>
  `,
  styles: [`
    .volume { display: flex; align-items: center; gap: 8px; }
    .vol-icon { width: 18px; height: 18px; color: var(--text-secondary); }
    .volume-bar { width: 90px; height: 4px; border-radius: var(--radius-full); background: var(--bg-elevated); cursor: pointer; position: relative; }
    .volume-bar:hover { height: 6px; }
    .volume-fill { height: 100%; border-radius: inherit; background: var(--accent-primary); }
  `]
})
export class VolumeControlComponent {
  @Input() volumePct = 80;
  @Output() volume = new EventEmitter<number>();

  onClick(event: MouseEvent): void {
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    this.volume.emit((event.clientX - rect.left) / rect.width);
  }
}
