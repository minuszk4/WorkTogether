import { TestBed, ComponentFixture, fakeAsync, tick } from '@angular/core/testing';
import { VoicePillComponent } from './voice-pill.component';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { Subject } from 'rxjs';

describe('VoicePillComponent', () => {
  let component: VoicePillComponent;
  let fixture: ComponentFixture<VoicePillComponent>;
  let mockChatWsService: any;
  let liveReactionSubject: Subject<any>;

  beforeEach(async () => {
    liveReactionSubject = new Subject<any>();
    mockChatWsService = {
      liveReaction$: liveReactionSubject
    };

    await TestBed.configureTestingModule({
      imports: [VoicePillComponent],
      providers: [
        { provide: ChatWsService, useValue: mockChatWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(VoicePillComponent);
    component = fixture.componentInstance;
    component.p = {
      sid: 'sid1',
      identity: 'user123',
      display_name: 'David',
      isSpeaking: false,
      isMuted: false,
      isCurrentUser: false
    };
    fixture.detectChanges();
  });

  it('should triggers reaction bounce class and floating emojis on matching user rx event', fakeAsync(() => {
    expect(component.isReacting).toBeFalse();
    expect(component.floaters.length).toBe(0);

    liveReactionSubject.next({ user_id: 'user123', emoji: '🔥' });
    fixture.detectChanges();

    expect(component.isReacting).toBeTrue();
    expect(component.floaters.length).toBe(1);
    expect(component.floaters[0].emoji).toBe('🔥');

    tick(1200);
    expect(component.isReacting).toBeFalse();

    tick(300);
    expect(component.floaters.length).toBe(0);
  }));

  it('should ignore live reactions that do not match the participant identity', () => {
    liveReactionSubject.next({ user_id: 'other_user', emoji: '❤️' });
    fixture.detectChanges();

    expect(component.isReacting).toBeFalse();
    expect(component.floaters.length).toBe(0);
  });
});
