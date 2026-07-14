import { ComponentFixture, TestBed } from '@angular/core/testing';
import { WhiteboardComponent } from './whiteboard.component';
import { StateService } from '../../../../core/services/state.service';

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static readonly OPEN = 1;
  readyState = FakeWebSocket.OPEN;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  sent: string[] = [];

  constructor(public url: string) { FakeWebSocket.instances.push(this); }
  send(message: string): void { this.sent.push(message); }
  close(): void {}
}

describe('WhiteboardComponent', () => {
  let fixture: ComponentFixture<WhiteboardComponent>;
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    (globalThis as any).WebSocket = FakeWebSocket;
    FakeWebSocket.instances = [];

    TestBed.configureTestingModule({
      imports: [WhiteboardComponent],
      providers: [{ provide: StateService, useValue: { accessToken: 'token' } }]
    });
    fixture = TestBed.createComponent(WhiteboardComponent);
    fixture.componentInstance.roomId = 'room-1';
  });

  afterEach(() => (globalThis as any).WebSocket = originalWebSocket);

  it('replays a snapshot and sends completed strokes through the collaboration socket', () => {
    fixture.detectChanges();
    const socket = FakeWebSocket.instances[0];
    const canvas = fixture.nativeElement.querySelector('canvas') as HTMLCanvasElement;
    const context = canvas.getContext('2d')!;
    spyOn(context, 'stroke');

    socket.onmessage?.(new MessageEvent('message', { data: JSON.stringify({
      event: 'whiteboard.snapshot',
      payload: { operations: [{ type: 'stroke', points: [{ x: 1, y: 2 }, { x: 3, y: 4 }], color: '#000000', width: 2 }] }
    }) }));

    fixture.componentInstance.onPointerDown({ offsetX: 10, offsetY: 10, pointerId: 1 } as PointerEvent);
    fixture.componentInstance.onPointerUp({ offsetX: 20, offsetY: 20, pointerId: 1 } as PointerEvent);

    expect(context.stroke).toHaveBeenCalled();
    expect(JSON.parse(socket.sent[0])).toEqual(jasmine.objectContaining({ event: 'whiteboard.operation', room_id: 'room-1' }));
  });
});
