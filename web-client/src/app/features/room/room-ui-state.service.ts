import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export type StageMode = 'music-only' | 'music-voice' | 'screenshare' | 'video';
export type RoomMode = 'chill' | 'focus' | 'collaborate';

export interface RoomUIState {
  roomMode: RoomMode;
  stageMode: StageMode;
  isChatOpen: boolean;
  isQueueOpen: boolean;
  isSubroomsOpen: boolean;
  isNotesOpen: boolean;
  isTimerOpen: boolean;
  isLyricsOpen: boolean;
  isBookmarksOpen: boolean;
  unreadCount: number;
  isIdentityOpen: boolean;
  isStatsOpen: boolean;
}

const VALID_MODES: StageMode[] = ['music-only', 'music-voice', 'screenshare', 'video'];
const ROOM_MODE_PRESETS: Record<RoomMode, Pick<RoomUIState, 'isChatOpen' | 'isQueueOpen' | 'isNotesOpen' | 'isTimerOpen'>> = {
  chill: { isChatOpen: false, isQueueOpen: true, isNotesOpen: false, isTimerOpen: false },
  focus: { isChatOpen: false, isQueueOpen: false, isNotesOpen: true, isTimerOpen: true },
  collaborate: { isChatOpen: true, isQueueOpen: false, isNotesOpen: true, isTimerOpen: false }
};

/** Local Room UI state, isolated from app-wide StateService. */
@Injectable({ providedIn: 'root' })
export class RoomUiStateService {
  private readonly state$ = new BehaviorSubject<RoomUIState>({
    roomMode: 'chill',
    stageMode: 'music-only',
    isChatOpen: false,
    isQueueOpen: true,
    isSubroomsOpen: false,
    isNotesOpen: false,
    isTimerOpen: false,
    isLyricsOpen: false,
    isBookmarksOpen: false,
    unreadCount: 0,
    isIdentityOpen: false,
    isStatsOpen: false
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

  public toggleSubrooms(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isSubroomsOpen : open;
    this.patch({ isSubroomsOpen: next });
  }

  public toggleNotes(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isNotesOpen : open;
    this.patch({ isNotesOpen: next });
  }

  public toggleTimer(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isTimerOpen : open;
    this.patch({ isTimerOpen: next });
  }

  public toggleLyrics(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isLyricsOpen : open;
    this.patch({ isLyricsOpen: next });
  }

  public toggleBookmarks(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isBookmarksOpen : open;
    this.patch({ isBookmarksOpen: next });
  }

  public toggleIdentity(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isIdentityOpen : open;
    this.patch({ isIdentityOpen: next });
  }

  public toggleStats(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isStatsOpen : open;
    this.patch({ isStatsOpen: next });
  }

  public markUnread(n: number): void {
    this.patch({ unreadCount: this.state$.value.unreadCount + n });
  }

  public setStageMode(mode: StageMode): void {
    if (!VALID_MODES.includes(mode)) return;
    this.patch({ stageMode: mode });
  }

  public applyRoomMode(mode: RoomMode): void {
    if (!Object.prototype.hasOwnProperty.call(ROOM_MODE_PRESETS, mode)) return;
    this.patch({ roomMode: mode, ...ROOM_MODE_PRESETS[mode] });
  }

  private patch(partial: Partial<RoomUIState>): void {
    this.state$.next({ ...this.state$.value, ...partial });
  }
}
