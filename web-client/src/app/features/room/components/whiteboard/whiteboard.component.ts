import { AfterViewInit, Component, ElementRef, Input, OnDestroy, ViewChild, inject } from '@angular/core';
import { StateService } from '../../../../core/services/state.service';

@Component({ selector: 'app-whiteboard', standalone: true, template: '<canvas #canvas width="800" height="420" (pointerdown)="onPointerDown($event)" (pointerup)="onPointerUp($event)"></canvas>' })
export class WhiteboardComponent implements AfterViewInit, OnDestroy {
  @Input({ required: true }) roomId = '';
  @ViewChild('canvas') canvas!: ElementRef<HTMLCanvasElement>;
  private state = inject(StateService); private socket?: WebSocket; private start?: { x: number; y: number };
  ngAfterViewInit(): void { this.socket = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/v1/rooms/${this.roomId}/collab/ws?token=${encodeURIComponent(this.state.accessToken)}`); this.socket.onmessage = e => this.receive(JSON.parse(e.data)); }
  ngOnDestroy(): void { this.socket?.close(); }
  onPointerDown(e: PointerEvent): void { this.start = { x: e.offsetX, y: e.offsetY }; }
  onPointerUp(e: PointerEvent): void { if (!this.start) return; const operation = { type: 'stroke', points: [this.start, { x: e.offsetX, y: e.offsetY }], color: '#000000', width: 2 }; this.draw(operation); this.socket?.send(JSON.stringify({ event: 'whiteboard.operation', room_id: this.roomId, payload: operation })); this.start = undefined; }
  private receive(message: any): void { if (message.event === 'whiteboard.snapshot') for (const op of message.payload.operations) this.draw(op); if (message.event === 'whiteboard.operation') this.draw(message.payload); }
  private draw(op: any): void { const c = this.canvas.nativeElement.getContext('2d')!; c.strokeStyle = op.color; c.lineWidth = op.width; c.beginPath(); c.moveTo(op.points[0].x, op.points[0].y); for (const p of op.points.slice(1)) c.lineTo(p.x, p.y); c.stroke(); }
}
