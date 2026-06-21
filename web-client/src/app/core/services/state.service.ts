import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export interface DashboardSectionState {
  roomsReady: boolean;
  friendsReady: boolean;
  pendingReady: boolean;
  notificationsReady: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class StateService {
  public user$ = new BehaviorSubject<any | null>(null);
  public accessToken$ = new BehaviorSubject<string>('');
  public activeRoom$ = new BehaviorSubject<any | null>(null);
  public activeRoomMembers$ = new BehaviorSubject<any[]>([]);
  public roomMemberRole$ = new BehaviorSubject<string>('MEMBER');
  public roomPermissions$ = new BehaviorSubject<string[]>([]);
  public friends$ = new BehaviorSubject<any[]>([]);
  public notifications$ = new BehaviorSubject<any[]>([]);
  public roomList$ = new BehaviorSubject<any[]>([]);
  public voiceConnected$ = new BehaviorSubject<boolean>(false);
  public voiceParticipants$ = new BehaviorSubject<Map<string, any>>(new Map());
  public unreadNotificationsCount$ = new BehaviorSubject<number>(0);
  public isVideoHidden$ = new BehaviorSubject<boolean>(false);
  public isVoiceFocused$ = new BehaviorSubject<boolean>(false);
  public dashboardSectionState$ = new BehaviorSubject<DashboardSectionState>({
    roomsReady: false,
    friendsReady: false,
    pendingReady: false,
    notificationsReady: false
  });

  constructor() {
    // Restore user from local storage if exists
    const savedUser = localStorage.getItem('user');
    if (savedUser) {
      try {
        this.user$.next(JSON.parse(savedUser));
      } catch (e) {
        localStorage.removeItem('user');
      }
    }

    // Save user to local storage on change
    this.user$.subscribe(user => {
      if (user) {
        localStorage.setItem('user', JSON.stringify(user));
      } else {
        localStorage.removeItem('user');
      }
    });
  }

  public get user() { return this.user$.value; }
  public get accessToken() { return this.accessToken$.value; }
  public get activeRoom() { return this.activeRoom$.value; }
  public get activeRoomMembers() { return this.activeRoomMembers$.value; }
  public get roomMemberRole() { return this.roomMemberRole$.value; }
  public get roomPermissions() { return this.roomPermissions$.value; }
  public get friends() { return this.friends$.value; }
  public get notifications() { return this.notifications$.value; }
  public get roomList() { return this.roomList$.value; }
  public get voiceConnected() { return this.voiceConnected$.value; }
  public get voiceParticipants() { return this.voiceParticipants$.value; }
  public get unreadNotificationsCount() { return this.unreadNotificationsCount$.value; }
  public get isVideoHidden() { return this.isVideoHidden$.value; }
  public get isVoiceFocused() { return this.isVoiceFocused$.value; }
  public get dashboardSectionState() { return this.dashboardSectionState$.value; }

  public setDashboardSectionReady(section: keyof DashboardSectionState, ready = true): void {
    this.dashboardSectionState$.next({
      ...this.dashboardSectionState$.value,
      [section]: ready
    });
  }

  public resetDashboardSectionState(): void {
    this.dashboardSectionState$.next({
      roomsReady: false,
      friendsReady: false,
      pendingReady: false,
      notificationsReady: false
    });
  }

  public clear(): void {
    this.user$.next(null);
    this.accessToken$.next('');
    this.activeRoom$.next(null);
    this.activeRoomMembers$.next([]);
    this.roomMemberRole$.next('MEMBER');
    this.roomPermissions$.next([]);
    this.friends$.next([]);
    this.notifications$.next([]);
    this.roomList$.next([]);
    this.voiceConnected$.next(false);
    this.voiceParticipants$.next(new Map());
    this.unreadNotificationsCount$.next(0);
    this.isVideoHidden$.next(false);
    this.isVoiceFocused$.next(false);
    this.resetDashboardSectionState();
  }
}
