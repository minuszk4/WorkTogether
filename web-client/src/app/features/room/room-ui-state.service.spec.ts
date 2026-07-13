import { TestBed } from '@angular/core/testing';
import { RoomUiStateService } from './room-ui-state.service';

describe('RoomUiStateService', () => {
  let svc: RoomUiStateService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    svc = TestBed.inject(RoomUiStateService);
  });

  it('starts closed with music-only stage and zero unread', () => {
    expect(svc.uiState.isChatOpen).toBeFalse();
    expect(svc.uiState.isQueueOpen).toBeTrue();
    expect(svc.uiState.stageMode).toBe('music-only');
    expect(svc.uiState.unreadCount).toBe(0);
  });

  it('toggles chat and resets unread when opened', () => {
    svc.markUnread(3);
    expect(svc.uiState.unreadCount).toBe(3);

    svc.toggleChat(true);
    expect(svc.uiState.isChatOpen).toBeTrue();
    expect(svc.uiState.unreadCount).toBe(0);
  });

  it('setStageMode ignores unknown mode', () => {
    svc.setStageMode('screenshare');
    expect(svc.uiState.stageMode).toBe('screenshare');
    // @ts-expect-error invalid mode
    svc.setStageMode('nope');
    expect(svc.uiState.stageMode).toBe('screenshare');
  });

  it('applies focused and collaborative room presets without changing the stage', () => {
    svc.setStageMode('screenshare');
    svc.applyRoomMode('focus');
    expect(svc.uiState.roomMode).toBe('focus');
    expect(svc.uiState.isTimerOpen).toBeTrue();
    expect(svc.uiState.isNotesOpen).toBeTrue();
    expect(svc.uiState.isQueueOpen).toBeFalse();
    expect(svc.uiState.stageMode).toBe('screenshare');

    svc.applyRoomMode('collaborate');
    expect(svc.uiState.isChatOpen).toBeTrue();
    expect(svc.uiState.isNotesOpen).toBeTrue();
    expect(svc.uiState.isTimerOpen).toBeFalse();
  });
});
