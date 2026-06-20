import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** App button. Variants: 'primary' (gradient), 'secondary' (glass), 'ghost'. */
@Component({
  selector: 'app-ocean-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      type="button"
      class="ocean-btn"
      [class.primary]="variant === 'primary'"
      [class.secondary]="variant === 'secondary'"
      [class.ghost]="variant === 'ghost'"
      [disabled]="disabled"
      (click)="onClick()"
    >
      <ng-content></ng-content>
    </button>
  `,
  styles: [`
    .ocean-btn {
      font-family: var(--font-sans); font-weight: 600;
      padding: 10px 18px; border: none; border-radius: var(--radius-full);
      cursor: pointer; transition: var(--transition-fast);
      color: var(--text-primary);
    }
    .ocean-btn.primary {
      background: var(--accent-gradient); color: #05121f;
      box-shadow: var(--shadow-accent);
    }
    .ocean-btn.primary:hover:not(:disabled) { transform: translateY(-1px); }
    .ocean-btn.secondary {
      background: var(--bg-glass); backdrop-filter: blur(12px);
      border: 1px solid var(--border-color);
    }
    .ocean-btn.secondary:hover:not(:disabled) { border-color: var(--border-focus); }
    .ocean-btn.ghost { background: transparent; color: var(--text-secondary); }
    .ocean-btn.ghost:hover:not(:disabled) { color: var(--text-primary); background: var(--bg-hover); }
    .ocean-btn:active:not(:disabled) { transform: scale(0.97); }
    .ocean-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  `]
})
export class OceanButtonComponent {
  @Input() variant: 'primary' | 'secondary' | 'ghost' = 'primary';
  @Input() disabled = false;
  @Output() clicked = new EventEmitter<void>();

  onClick(): void {
    if (!this.disabled) this.clicked.emit();
  }
}
