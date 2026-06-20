import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export type StageMode = 'music-only' | 'music-voice' | 'screenshare' | 'video';

export interface RoomUIState {
  stageMode: StageMode;
  isChatOpen: boolean;
  isQueueOpen: boolean;
  unreadCount: number;
}

const VALID_MODES: StageMode[] = ['music-only', 'music-voice', 'screenshare', 'video'];

/** Local Room UI state, isolated from app-wide StateService. */
@Injectable({ providedIn: 'root' })
export class RoomUiStateService {
  private readonly state$ = new BehaviorSubject<RoomUIState>({
    stageMode: 'music-only',
    isChatOpen: false,
    isQueueOpen: true,
    unreadCount: 0
  });

  public get uiState(): RoomUIState {
    return this.state$.value;
  }

  public get changes$() {
    return this.state$.asObservable();
  }

  public toggleChat(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isChatOpen : open;
    this.patch({ isChatOpen: next, unreadCount: next ? 0 : this.state$.value.unreadCount });
  }

  public toggleQueue(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isQueueOpen : open;
    this.patch({ isQueueOpen: next });
  }

  public markUnread(n: number): void {
    this.patch({ unreadCount: this.state$.value.unreadCount + n });
  }

  public setStageMode(mode: StageMode): void {
    if (!VALID_MODES.includes(mode)) return;
    this.patch({ stageMode: mode });
  }

  private patch(partial: Partial<RoomUIState>): void {
    this.state$.next({ ...this.state$.value, ...partial });
  }
}
