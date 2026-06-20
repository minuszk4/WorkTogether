import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

@Component({
  selector: 'app-room-quick-reactions',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="quick-reactions-bar" role="toolbar" aria-label="Bắn biểu cảm">
      @for (emoji of emojis; track emoji) {
        <button (click)="react(emoji)" [attr.aria-label]="'Bắn ' + emoji">{{ emoji }}</button>
      }
    </div>
  `,
  styleUrl: './quick-reactions.component.css'
})
export class QuickReactionsComponent {
  private chatWs = inject(ChatWsService);
  public emojis = ['❤️', '🔥', '👏', '😮', '📚'];

  react(emoji: string) {
    this.chatWs.sendLiveReaction(emoji);
  }
}
