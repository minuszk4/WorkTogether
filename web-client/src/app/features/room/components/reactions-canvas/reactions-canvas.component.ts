import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

interface FloatingReaction {
  id: number;
  emoji: string;
  leftPercent: number;
}

@Component({
  selector: 'app-room-reactions-canvas',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="reactions-canvas" aria-hidden="true">
      @for (r of reactions; track r.id) {
        <span class="flying-emoji" [style.left.%]="r.leftPercent">{{ r.emoji }}</span>
      }
    </div>
  `,
  styleUrl: './reactions-canvas.component.css'
})
export class ReactionsCanvasComponent implements OnInit, OnDestroy {
  private chatWs = inject(ChatWsService);
  private sub: Subscription | null = null;

  public reactions: FloatingReaction[] = [];
  private nextId = 0;

  ngOnInit(): void {
    this.sub = this.chatWs.liveReaction$.subscribe(rx => {
      if (rx) {
        this.spawnReaction(rx.emoji);
      }
    });
  }

  ngOnDestroy(): void {
    if (this.sub) this.sub.unsubscribe();
  }

  private spawnReaction(emoji: string) {
    const id = this.nextId++;
    const leftPercent = Math.floor(Math.random() * 80) + 10;
    
    this.reactions.push({ id, emoji, leftPercent });
    
    setTimeout(() => {
      this.reactions = this.reactions.filter(r => r.id !== id);
    }, 3000);
  }
}
