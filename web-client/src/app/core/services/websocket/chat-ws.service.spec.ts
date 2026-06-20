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
});
