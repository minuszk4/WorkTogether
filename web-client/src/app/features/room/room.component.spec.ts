import { fakeAsync, TestBed, tick } from '@angular/core/testing';
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
import { ActivatedRoute, convertToParamMap } from '@angular/router';

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
	  user: {
		updateStatus: () => of({})
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
      pinnedUpdate$: new BehaviorSubject(null),
      listenerStates$: new BehaviorSubject({}),
      liveReaction$: new BehaviorSubject(null),
      roomVibe$: new BehaviorSubject(null),
      roomMode$: new BehaviorSubject(null)
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
	  activeSpeakers$: new BehaviorSubject([]),
	  connectionState$: new BehaviorSubject('disconnected')
    };
    mockToastService = {
      success: () => {},
      error: () => {},
      info: () => {}
    };
    mockRoomUiStateService = {
      changes$: new BehaviorSubject({ roomMode: 'chill', isChatOpen: false, isQueueOpen: true, stageMode: 'music-only', unreadCount: 0, isIdentityOpen: false, isStatsOpen: false }),
      uiState: { roomMode: 'chill', isChatOpen: false, isQueueOpen: true, stageMode: 'music-only', unreadCount: 0, isIdentityOpen: false, isStatsOpen: false },
      setStageMode: () => {},
      applyRoomMode: () => {}
    };

    TestBed.configureTestingModule({
      imports: [RouterTestingModule, RoomComponent],
      providers: [
		{
		  provide: ActivatedRoute,
		  useValue: { snapshot: { paramMap: convertToParamMap({ room_id: 'room-123' }) } }
		},
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

  it('does not join voice or request microphone permission when the room opens', fakeAsync(() => {
	const getToken = spyOn(mockApiService.voice, 'getToken').and.callThrough();
	const fixture = TestBed.createComponent(RoomComponent);
	fixture.componentInstance.ngOnInit();
	tick(1000);

	expect(getToken).not.toHaveBeenCalled();
	fixture.componentInstance.ngOnDestroy();
  }));
});
