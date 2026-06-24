import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { VoiceService } from '../../../../core/services/voice.service';
import { StateService } from '../../../../core/services/state.service';
import { VoicePillComponent, PillParticipant } from './voice-pill.component';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';

@Component({
  selector: 'app-room-voice-pills',
  standalone: true,
  imports: [CommonModule, VoicePillComponent],
  template: `
    @if (pills.length > 0) {
      <div class="voice-pills" role="list">
        @for (p of pills; track p.sid) {
          <app-room-voice-pill role="listitem" [p]="p"></app-room-voice-pill>
        }
      </div>
    }
  `,
  styleUrl: './voice-pills.component.css'
})
export class VoicePillsComponent implements OnInit, OnDestroy {
  private voiceService = inject(VoiceService);
  private state = inject(StateService);
  private chatWs = inject(ChatWsService);
  private playbackWs = inject(PlaybackWsService);

  public pills: PillParticipant[] = [];
  private rawParticipants: any[] = [];
  private members: any[] = [];
  private activeSpeakers: string[] = [];
  private listenerStates: Record<string, { is_playing: boolean; position_ms: number; updated_at: number }> = {};
  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.voiceService.participants$.subscribe(list => { this.rawParticipants = list || []; this.sync(); }),
      this.voiceService.activeSpeakers$.subscribe(s => { this.activeSpeakers = s || []; this.sync(); }),
      this.state.activeRoomMembers$.subscribe(m => { this.members = m || []; this.sync(); }),
      this.chatWs.listenerStates$.subscribe(states => { this.listenerStates = states || {}; this.sync(); }),
      this.playbackWs.playbackSync$.subscribe(() => { this.sync(); })
    );
  }

  ngOnDestroy(): void { this.subs.forEach(s => s.unsubscribe()); }

  private sync(): void {
    const master = this.playbackWs.playbackSync$.value;
    const now = Date.now();

    const getMemberPresence = (identity: string) => {
      const state = this.listenerStates[identity];
      if (!state) return { isPlaying: undefined, isUnsynced: false };
      
      const isPlaying = state.is_playing;
      let isUnsynced = false;
      if (master && master.state === 'playing') {
        const estimatedPosition = state.is_playing
          ? state.position_ms + (now - state.updated_at)
          : state.position_ms;
        const diff = Math.abs(estimatedPosition - master.position_ms);
        isUnsynced = diff > 3000;
      }
      return { isPlaying, isUnsynced };
    };

    const live = this.rawParticipants.map(p => {
      const member = this.members.find(m => m.user_id === p.identity || m.id === p.identity);
      const presence = getMemberPresence(p.identity || '');
      return {
        sid: p.sid,
        display_name: member?.display_name || p.identity || 'Unknown',
        avatar_url: member?.avatar_url || '',
        isSpeaking: this.activeSpeakers.includes(p.sid),
        isMuted: !!p.isMuted,
        isCurrentUser: !!member?.isCurrentUser || !!p.isLocal,
        isPlaying: presence.isPlaying,
        isUnsynced: presence.isUnsynced
      } as PillParticipant;
    });
    
    const liveIds = new Set(this.rawParticipants.map(p => p.identity).filter(Boolean));
    const silent = this.members
      .filter(m => !liveIds.has(m.user_id))
      .map(m => {
        const presence = getMemberPresence(m.user_id || m.id || '');
        return {
          sid: `member-${m.user_id}`,
          identity: m.user_id || m.id,
          display_name: m.display_name || m.username || 'Thành viên',
          avatar_url: m.avatar_url || '',
          isSpeaking: false,
          isMuted: false,
          isCurrentUser: !!m.isCurrentUser,
          isPlaying: presence.isPlaying,
          isUnsynced: presence.isUnsynced
        } as PillParticipant;
      });
    this.pills = [...live, ...silent].sort((a, b) => this.weight(b) - this.weight(a));
  }

  private weight(p: PillParticipant): number {
    return (p.isSpeaking ? 1000 : 0) + (p.sid.startsWith('member-') ? 0 : 500) + (p.isCurrentUser ? 1 : 0);
  }
}
