import { TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { RoomComponent } from './room.component';
import { ApiService } from '../../core/services/api.service';
import { StateService } from '../../core/services/state.service';
import { ChatWsService } from '../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../core/services/voice.service';
import { ToastService } from '../../shared/services/toast.service';
import { RoomUiStateService } from './room-ui-state.service';
import { BehaviorSubject, of } from 'rxjs';

describe('RoomComponent', () => {
  let mockApiService: any;
  let mockStateService: any;
  let mockChatWsService: any;
  let mockPlaybackWsService: any;
  let mockVoiceService: any;
  let mockToastService: any;
  let mockRoomUiStateService: any;

  beforeEach(() => {
    mockApiService = {
      room: {
        get: () => of({ name: 'Test Room', description: 'Test Desc' }),
        listMembers: () => of([])
      },
      voice: {
        getToken: () => of({ livekit_url: 'ws://livekit', token: 'token' })
      }
    };
    mockStateService = {
      activeRoom$: new BehaviorSubject(null),
      activeRoomMembers$: new BehaviorSubject([]),
      roomMemberRole$: new BehaviorSubject('MEMBER'),
      roomPermissions$: new BehaviorSubject([]),
      accessToken: 'test-token',
      user: { id: 'user-123', display_name: 'Test User' }
    };
    mockChatWsService = {
      connect: () => {},
      disconnect: () => {},
      messageReceived$: new BehaviorSubject(null),
      messageDeleted$: new BehaviorSubject(null),
      pinnedUpdate$: new BehaviorSubject(null)
    };
    mockPlaybackWsService = {
      connect: () => {},
      disconnect: () => {},
      playbackSync$: new BehaviorSubject(null)
    };
    mockVoiceService = {
      connect: () => Promise.resolve(),
      disconnect: () => {},
      connected$: new BehaviorSubject(false),
      isMuted$: new BehaviorSubject(false),
      isScreenSharing$: new BehaviorSubject(false),
      isCameraActive$: new BehaviorSubject(false),
      participants$: new BehaviorSubject([]),
      activeSpeakers$: new BehaviorSubject([])
    };
    mockToastService = {
      success: () => {},
      error: () => {},
      info: () => {}
    };
    mockRoomUiStateService = {
      changes$: new BehaviorSubject({ isChatOpen: false, isQueueOpen: true, stageMode: 'music-only', unreadCount: 0 }),
      uiState: { isChatOpen: false, isQueueOpen: true, stageMode: 'music-only', unreadCount: 0 },
      setStageMode: () => {}
    };

    TestBed.configureTestingModule({
      imports: [RouterTestingModule, RoomComponent],
      providers: [
        { provide: ApiService, useValue: mockApiService },
        { provide: StateService, useValue: mockStateService },
        { provide: ChatWsService, useValue: mockChatWsService },
        { provide: PlaybackWsService, useValue: mockPlaybackWsService },
        { provide: VoiceService, useValue: mockVoiceService },
        { provide: ToastService, useValue: mockToastService },
        { provide: RoomUiStateService, useValue: mockRoomUiStateService }
      ]
    });
  });

  it('creates', () => {
    const fixture = TestBed.createComponent(RoomComponent);
    expect(fixture.componentInstance).toBeTruthy();
  });
});
