import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/**
 * Icon-only button. Always sets an accessible label via `label`.
 * Variants: 'default' | 'active' | 'danger'.
 */
@Component({
  selector: 'app-icon-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      type="button"
      class="icon-btn"
      [class.active]="variant === 'active'"
      [class.danger]="variant === 'danger'"
      [disabled]="disabled"
      [attr.aria-label]="label"
      [attr.title]="label"
      (click)="onClick()"
    >
      <ng-content></ng-content>
    </button>
  `,
  styles: [`
    .icon-btn {
      width: 40px; height: 40px;
      display: inline-flex; align-items: center; justify-content: center;
      border: none; border-radius: var(--radius-md);
      background: transparent; color: var(--text-secondary);
      cursor: pointer; transition: var(--transition-fast);
    }
    .icon-btn:hover:not(:disabled) {
      background: var(--bg-hover); color: var(--text-primary); transform: scale(1.05);
    }
    .icon-btn:active:not(:disabled) { transform: scale(0.96); }
    .icon-btn.active {
      background: var(--accent-dim); color: var(--accent-primary);
      box-shadow: inset 0 0 0 1px rgba(45, 212, 191, 0.25);
    }
    .icon-btn.danger { color: var(--danger); }
    .icon-btn:disabled { opacity: 0.45; cursor: not-allowed; }
    ::ng-deep .icon-btn svg { width: 20px; height: 20px; }
  `]
})
export class IconButtonComponent {
  @Input() label = '';
  @Input() disabled = false;
  @Input() variant: 'default' | 'active' | 'danger' = 'default';
  @Output() clicked = new EventEmitter<void>();

  onClick(): void {
    if (!this.disabled) {
      this.clicked.emit();
    }
  }
}
