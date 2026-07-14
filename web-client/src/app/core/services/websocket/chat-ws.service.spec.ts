import { TestBed } from '@angular/core/testing';
import { ChatWsService } from './chat-ws.service';

describe('ChatWsService', () => {
  let service: ChatWsService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(ChatWsService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should have presence streams and helpers', (done) => {
    // These streams do not exist yet, causing a compilation failure
    expect(service.listenerStates$).toBeDefined();
    expect(service.liveReaction$).toBeDefined();
    expect(service.roomVibe$).toBeDefined();

    service.listenerStates$.subscribe((states: any) => {
      if (!states || Object.keys(states).length === 0) return;
      expect(states['user-1']).toBeDefined();
      expect(states['user-1'].is_playing).toBeTrue();
      done();
    });

    // Simulate event message handler call
    const handler = (service as any).handleMessage.bind(service);
    handler(JSON.stringify({
      event: 'presence:listener_states',
      payload: {
        user_states: {
          'user-1': { is_playing: true, position_ms: 1000, updated_at: Date.now() }
        }
      }
    }));
  });

  it('emits synchronized room mode changes', (done) => {
    service.roomMode$.subscribe((change) => {
      if (!change) return;
      expect(change.mode).toBe('focus');
      expect(change.changed_by).toBe('moderator-1');
      done();
    });

    (service as any).handleMessage(JSON.stringify({
      event: 'room:mode_changed',
      payload: { mode: 'focus', changed_by: 'moderator-1' }
    }));
  });

  it('emits shared session events', (done) => {
    service.roomEvent$.subscribe((event) => {
      expect(event.event).toBe('session:agenda_added');
      expect(event.payload.id).toBe('agenda-1');
      done();
    });

    (service as any).handleMessage(JSON.stringify({
      event: 'session:agenda_added',
      payload: { id: 'agenda-1' }
    }));
  });

  it('emits late translations independently from the original chat message', (done) => {
    service.translationReceived$.subscribe((translation) => {
      expect(translation.event_id).toBe('message-1');
      expect(translation.text).toBe('xin chào');
      expect(translation.kind).toBe('chat');
      done();
    });

    (service as any).handleMessage(JSON.stringify({
      event: 'translation:received',
      payload: { event_id: 'message-1', text: 'xin chào', kind: 'chat', target_language: 'vi' }
    }));
  });

  it('emits room subtitles separately from chat and playback events', (done) => {
    service.subtitleReceived$.subscribe((subtitle) => {
      expect(subtitle.id).toBe('subtitle-1');
      expect(subtitle.text).toBe('Hello everyone');
      done();
    });

    (service as any).handleMessage(JSON.stringify({
      event: 'subtitle:received',
      payload: { id: 'subtitle-1', sender_id: 'user-1', text: 'Hello everyone', language: 'en' }
    }));
  });
});
