import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ToastService } from '../../services/toast.service';

@Component({
  selector: 'app-toast',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div id="toast-container">
      @for (toast of toasts$ | async; track toast.id) {
        <div [class]="'toast toast-' + toast.type" (click)="remove(toast.id)">
          <span class="toast-icon">
            @if (toast.type === 'success') { ✓ }
            @else if (toast.type === 'error') { ✗ }
            @else { ℹ }
          </span>
          <span class="toast-text">{{ toast.text }}</span>
        </div>
      }
    </div>
  `
})
export class ToastComponent {
  private toastService = inject(ToastService);
  public toasts$ = this.toastService.toasts$;

  public remove(id: string): void {
    this.toastService.remove(id);
  }
}
