import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

@Component({
  selector: 'app-room-vibe-meter',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="vibe-meter" [class]="'vibe-' + currentVibe" role="status" aria-live="polite">
      <span class="vibe-icon">{{ vibeEmoji }}</span>
      <span class="vibe-label">Phòng: <span class="vibe-value">{{ currentVibe }}</span></span>
    </div>
  `,
  styleUrl: './vibe-meter.component.css'
})
export class RoomVibeMeterComponent implements OnInit, OnDestroy {
  private chatWs = inject(ChatWsService);
  private sub: Subscription | null = null;

  public currentVibe = 'chill';

  get vibeEmoji(): string {
    switch (this.currentVibe) {
      case 'hype': return '🔥';
      case 'study': return '📚';
      default: return '🍃';
    }
  }

  ngOnInit(): void {
    this.sub = this.chatWs.roomVibe$.subscribe(v => {
      if (v && v.current_vibe) {
        this.currentVibe = v.current_vibe;
      }
    });
  }

  ngOnDestroy(): void {
    if (this.sub) this.sub.unsubscribe();
  }
}
