import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Frosted-glass container using --bg-glass + backdrop blur. */
@Component({
  selector: 'app-glass-panel',
  standalone: true,
  imports: [CommonModule],
  template: `<div class="glass-panel" [class.elevated]="elevated"><ng-content></ng-content></div>`,
  styles: [`
    .glass-panel {
      background: var(--bg-glass);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid var(--border-color);
      border-radius: var(--radius-lg);
    }
    .glass-panel.elevated { box-shadow: var(--shadow-lg); }
  `]
})
export class GlassPanelComponent {
  @Input() elevated = false;
}
