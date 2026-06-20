import { TestBed, ComponentFixture, fakeAsync, tick } from '@angular/core/testing';
import { ReactionsCanvasComponent } from './reactions-canvas.component';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { Subject } from 'rxjs';

describe('ReactionsCanvasComponent', () => {
  let component: ReactionsCanvasComponent;
  let fixture: ComponentFixture<ReactionsCanvasComponent>;
  let mockChatWsService: any;
  let liveReactionSubject: Subject<any>;

  beforeEach(async () => {
    liveReactionSubject = new Subject<any>();
    mockChatWsService = {
      liveReaction$: liveReactionSubject
    };

    await TestBed.configureTestingModule({
      imports: [ReactionsCanvasComponent],
      providers: [
        { provide: ChatWsService, useValue: mockChatWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ReactionsCanvasComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and render canvas container', () => {
    const element: HTMLElement = fixture.nativeElement;
    const container = element.querySelector('.reactions-canvas');
    expect(container).toBeDefined();
  });

  it('should append floating emoji elements when receiving live reaction events', fakeAsync(() => {
    expect(component.reactions.length).toBe(0);

    liveReactionSubject.next({ user_id: 'user1', emoji: '🔥' });
    fixture.detectChanges();

    expect(component.reactions.length).toBe(1);
    expect(component.reactions[0].emoji).toBe('🔥');

    const element: HTMLElement = fixture.nativeElement;
    const floatingEmoji = element.querySelector('.flying-emoji');
    expect(floatingEmoji).toBeDefined();
    expect(floatingEmoji?.textContent?.trim()).toBe('🔥');

    tick(3000);
    fixture.detectChanges();
    expect(component.reactions.length).toBe(0);
  }));
});
