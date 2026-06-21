import { Component, Input, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { Subscription, Subject } from 'rxjs';
import { debounceTime } from 'rxjs/operators';

export interface NoteBlock {
  id: string;
  note_id: string;
  block_type: 'text' | 'todo';
  content: string;
  is_checked: boolean;
  order_index: number;
  updated_by: string;
  updated_at: string;
}

@Component({
  selector: 'app-room-collab-notes',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './collab-notes.component.html',
  styleUrl: './collab-notes.component.css'
})
export class CollabNotesComponent implements OnInit, OnDestroy {
  public state = inject(StateService);
  private toast = inject(ToastService);

  @Input() roomId = '';

  private ws: WebSocket | null = null;
  public blocks: NoteBlock[] = [];
  public noteTitle = 'Ghi chú phòng';

  private subs: Subscription[] = [];
  private reconnectTimeout: any = null;

  // Debounce typing stream
  private blockUpdate$ = new Subject<NoteBlock>();

  ngOnInit(): void {
    this.connectWs();

    // Debounce content updates by 400ms to avoid flooding websocket
    this.subs.push(
      this.blockUpdate$.pipe(debounceTime(400)).subscribe(block => {
        this.sendCommand('block:update', {
          id: block.id,
          content: block.content,
          is_checked: block.is_checked
        });
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    this.disconnectWs();
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
    }
  }

  private connectWs(): void {
    if (!this.roomId) return;

    const token = this.state.accessToken;
    if (!token) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/rooms/${this.roomId}/collab/ws?token=${token}`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onmessage = (event) => {
      try {
        const rawMsg = JSON.parse(event.data);
        this.handleWsEvent(rawMsg.event, rawMsg.payload);
      } catch (err) {
        console.warn('Lỗi xử lý WS collab:', err);
      }
    };

    this.ws.onclose = () => {
      console.log('WS collab closed, reconnecting...');
      this.reconnectTimeout = setTimeout(() => this.connectWs(), 3000);
    };

    this.ws.onerror = (err) => {
      console.error('WS collab error:', err);
    };
  }

  private disconnectWs(): void {
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
  }

  private sendCommand(event: string, payload?: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    const msg = {
      event: event,
      room_id: this.roomId,
      payload: payload ? JSON.stringify(payload) : null
    };
    this.ws.send(JSON.stringify(msg));
  }

  private handleWsEvent(event: string, payload: any): void {
    switch (event) {
      case 'note:sync':
        if (payload) {
          this.blocks = payload.blocks || [];
          if (payload.note) {
            this.noteTitle = payload.note.title || 'Ghi chú phòng';
          }
          this.sortBlocks();
        }
        break;

      case 'block:created':
        if (payload) {
          // Prevent duplicate inserts
          const idx = this.blocks.findIndex(b => b.id === payload.id);
          if (idx === -1) {
            this.blocks.push(payload);
            this.sortBlocks();
          }
        }
        break;

      case 'block:updated':
        if (payload) {
          const idx = this.blocks.findIndex(b => b.id === payload.id);
          if (idx !== -1) {
            // Only update content if not currently focused by local user to prevent cursor jumping
            const el = document.activeElement;
            const isEditingLocally = el && el.id === `input-${payload.id}`;
            if (!isEditingLocally) {
              this.blocks[idx].content = payload.content;
            }
            this.blocks[idx].is_checked = payload.is_checked;
            this.blocks[idx].updated_by = payload.updated_by;
            this.blocks[idx].updated_at = payload.updated_at;
          }
        }
        break;

      case 'block:deleted':
        if (payload && payload.id) {
          this.blocks = this.blocks.filter(b => b.id !== payload.id);
        }
        break;

      case 'block:ordered':
        if (payload && payload.block_ids) {
          const idMap = new Map<string, number>(payload.block_ids.map((id: string, idx: number) => [id, idx]));
          this.blocks.forEach(b => {
            const newIdx = idMap.get(b.id);
            if (newIdx !== undefined) {
              b.order_index = newIdx;
            }
          });
          this.sortBlocks();
        }
        break;
    }
  }

  private sortBlocks(): void {
    this.blocks.sort((a, b) => a.order_index - b.order_index);
  }

  public onAddBlock(type: 'text' | 'todo'): void {
    const nextOrder = this.blocks.length > 0 ? this.blocks[this.blocks.length - 1].order_index + 1 : 0;
    const blockId = this.generateUUID();

    const newBlock: Partial<NoteBlock> = {
      id: blockId,
      block_type: type,
      content: '',
      is_checked: false,
      order_index: nextOrder
    };

    this.sendCommand('block:add', newBlock);
  }

  public onBlockChange(block: NoteBlock): void {
    // Push into debounce stream
    this.blockUpdate$.next(block);
  }

  public onCheckboxToggle(block: NoteBlock): void {
    // Checkboxes toggle immediately
    this.sendCommand('block:update', {
      id: block.id,
      content: block.content,
      is_checked: block.is_checked
    });
  }

  public onDeleteBlock(blockId: string): void {
    this.sendCommand('block:delete', { id: blockId });
  }

  public onMoveBlock(block: NoteBlock, direction: 'up' | 'down'): void {
    const idx = this.blocks.findIndex(b => b.id === block.id);
    if (idx === -1) return;

    if (direction === 'up' && idx > 0) {
      // Swap positions
      const temp = this.blocks[idx - 1];
      this.blocks[idx - 1] = this.blocks[idx];
      this.blocks[idx] = temp;
    } else if (direction === 'down' && idx < this.blocks.length - 1) {
      // Swap positions
      const temp = this.blocks[idx + 1];
      this.blocks[idx + 1] = this.blocks[idx];
      this.blocks[idx] = temp;
    } else {
      return; // No move possible
    }

    // Recalculate order indices
    this.blocks.forEach((b, i) => b.order_index = i);
    const blockIds = this.blocks.map(b => b.id);

    // Send ordering command
    this.sendCommand('block:move', { block_ids: blockIds });
  }

  private generateUUID(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }

  public getMemberName(userId: string): string {
    const list = this.state.activeRoomMembers$.value || [];
    const member = list.find(m => m.user_id === userId);
    return member?.display_name || userId.slice(0, 8);
  }
}
