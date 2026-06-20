import { TestBed, ComponentFixture } from '@angular/core/testing';
import { RoomVibeMeterComponent } from './vibe-meter.component';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { BehaviorSubject } from 'rxjs';

describe('RoomVibeMeterComponent', () => {
  let component: RoomVibeMeterComponent;
  let fixture: ComponentFixture<RoomVibeMeterComponent>;
  let mockChatWsService: any;
  let roomVibeSubject: BehaviorSubject<any>;

  beforeEach(async () => {
    roomVibeSubject = new BehaviorSubject<any>(null);
    mockChatWsService = {
      roomVibe$: roomVibeSubject
    };

    await TestBed.configureTestingModule({
      imports: [RoomVibeMeterComponent],
      providers: [
        { provide: ChatWsService, useValue: mockChatWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(RoomVibeMeterComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and display default chill vibe', () => {
    const element: HTMLElement = fixture.nativeElement;
    const label = element.querySelector('.vibe-label');
    expect(label?.textContent?.trim().toLowerCase()).toContain('chill');
  });

  it('should update visual styling and current vibe text when room vibe tick is received', () => {
    roomVibeSubject.next({
      current_vibe: 'hype',
      vibe_scores: { chill: 2, hype: 10, study: 1 }
    });
    fixture.detectChanges();

    expect(component.currentVibe).toBe('hype');

    const element: HTMLElement = fixture.nativeElement;
    const label = element.querySelector('.vibe-label');
    expect(label?.textContent?.trim().toLowerCase()).toContain('hype');

    const container = element.querySelector('.vibe-meter');
    expect(container?.classList.contains('vibe-hype')).toBeTrue();
  });
});
