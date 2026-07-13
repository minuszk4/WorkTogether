import { TestBed, ComponentFixture } from '@angular/core/testing';
import { VoicePillsComponent } from './voice-pills.component';
import { VoiceService } from '../../../../core/services/voice.service';
import { StateService } from '../../../../core/services/state.service';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';
import { BehaviorSubject } from 'rxjs';
import { provideHttpClient } from '@angular/common/http';

describe('VoicePillsComponent', () => {
  let component: VoicePillsComponent;
  let fixture: ComponentFixture<VoicePillsComponent>;

  let mockVoiceService: any;
  let mockStateService: any;
  let mockChatWsService: any;
  let mockPlaybackWsService: any;

  beforeEach(async () => {
    mockVoiceService = {
      participants$: new BehaviorSubject([]),
      activeSpeakers$: new BehaviorSubject([])
    };

    mockStateService = {
      activeRoomMembers$: new BehaviorSubject([
        { user_id: 'user1', display_name: 'Alice', avatar_url: '', isCurrentUser: false },
        { user_id: 'user2', display_name: 'Bob', avatar_url: '', isCurrentUser: false }
      ])
    };

    mockChatWsService = {
      listenerStates$: new BehaviorSubject({}),
      liveReaction$: new BehaviorSubject(null),
      roomVibe$: new BehaviorSubject(null)
    };

    mockPlaybackWsService = {
      playbackSync$: new BehaviorSubject(null)
    };

    await TestBed.configureTestingModule({
      imports: [VoicePillsComponent],
      providers: [
		provideHttpClient(),
        { provide: VoiceService, useValue: mockVoiceService },
        { provide: StateService, useValue: mockStateService },
        { provide: ChatWsService, useValue: mockChatWsService },
        { provide: PlaybackWsService, useValue: mockPlaybackWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(VoicePillsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and compute unsynced flags correctly', () => {
    mockPlaybackWsService.playbackSync$.next({
      state: 'playing',
      current_track: { id: 'track1', duration_ms: 60000 },
      position_ms: 10000,
      updated_at: Date.now()
    });

    mockChatWsService.listenerStates$.next({
      user1: { is_playing: true, position_ms: 9500, updated_at: Date.now() },
      user2: { is_playing: true, position_ms: 4000, updated_at: Date.now() }
    });

    (component as any).sync();
    fixture.detectChanges();

    const user1Pill = component.pills.find(p => p.display_name === 'Alice');
    const user2Pill = component.pills.find(p => p.display_name === 'Bob');

    expect(user1Pill).toBeDefined();
    expect(user1Pill?.isUnsynced).toBeFalse();

    expect(user2Pill).toBeDefined();
    expect(user2Pill?.isUnsynced).toBeTrue();
  });
});
