import { Component, OnDestroy, OnInit, inject, ViewChildren, QueryList, ElementRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { VoiceService } from '../../../../core/services/voice.service';
import { StateService } from '../../../../core/services/state.service';

@Component({
  selector: 'app-room-voice-grid',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './voice-grid.component.html',
  styleUrl: './voice-grid.component.css'
})
export class VoiceGridComponent implements OnInit, OnDestroy, AfterViewInit {
  @ViewChildren('voiceVideo') videoElements!: QueryList<ElementRef<HTMLVideoElement>>;
  public voiceService = inject(VoiceService);
  public state = inject(StateService);

  public isConnected = false;
  public participants: any[] = [];
  public activeSpeakers: string[] = [];
  public members: any[] = [];

  private rawParticipants: any[] = [];
  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.voiceService.connected$.subscribe(connected => {
        this.isConnected = connected;
        setTimeout(() => this.attachVideoTracks(), 0);
      }),
      this.voiceService.participants$.subscribe(list => {
        this.rawParticipants = list || [];
        this.syncParticipants();
        setTimeout(() => this.attachVideoTracks(), 0);
      }),
      this.voiceService.activeSpeakers$.subscribe(speakers => {
        this.activeSpeakers = speakers || [];
        this.syncParticipants();
      }),
      this.state.activeRoomMembers$.subscribe(members => {
        this.members = this.sortMembers(members || []);
        this.syncParticipants();
      })
    );
  }

  ngAfterViewInit(): void {
    this.subs.push(
      this.videoElements.changes.subscribe(() => {
        this.attachVideoTracks();
      })
    );
    this.attachVideoTracks();
  }

  ngOnDestroy(): void {
    this.subs.forEach(sub => sub.unsubscribe());

    // Detach all active video elements to release device camera/hardware
    if (this.videoElements) {
      this.videoElements.forEach(elRef => {
        const videoEl = elRef.nativeElement;
        const trackSid = videoEl.dataset['trackSid'];
        if (trackSid) {
          this.participants.forEach(p => {
            if (p.raw && typeof p.raw.getTrackPublications === 'function') {
              p.raw.getTrackPublications().forEach((pub: any) => {
                const track = pub.videoTrack || pub.track;
                if (track && track.sid === trackSid) {
                  try {
                    track.detach(videoEl);
                  } catch (e) {}
                }
              });
            }
          });
          videoEl.srcObject = null;
          delete videoEl.dataset['trackSid'];
        }
      });
    }
  }

  public get screenshareParticipant(): any {
    return this.participants.find(p => p.isScreenSharing) || null;
  }

  public isSpeaking(sid: string): boolean {
    return this.activeSpeakers.includes(sid);
  }

  public getInitials(label: string): string {
    if (!label) return 'WT';
    return label.slice(0, 2).toUpperCase();
  }

  public hasScreenshare(): boolean {
    return this.participants.some(participant => participant.isScreenSharing);
  }

  public getParticipantLabel(participant: any): string {
    const baseLabel = participant.display_name || participant.username || participant.identity || 'Unknown';
    return participant.isCurrentUser ? `${baseLabel} (Ban)` : baseLabel;
  }

  public getPresenceText(item: any): string {
    return item?.presence?.custom_text || item?.presence?.status || 'offline';
  }

  private syncParticipants(): void {
    const liveParticipants = this.rawParticipants
      .map(participant => {
        const member = this.getMemberByIdentity(participant.identity);
        return {
          ...participant,
          display_name: member?.display_name || participant.identity,
          username: member?.username || participant.identity,
          avatar_url: member?.avatar_url || '',
          presence: member?.presence || { status: 'offline', custom_text: '' },
          role: member?.role || 'MEMBER',
          permissions: member?.permissions || [],
          isCurrentUser: member?.isCurrentUser || participant.isLocal,
          isVoiceParticipant: true
        };
      });

    const liveMemberIds = new Set(
      liveParticipants
        .map(participant => participant.identity)
        .filter((identity: string) => !!identity)
    );

    const silentMembers = this.members
      .filter(member => !liveMemberIds.has(member.user_id))
      .map(member => ({
        sid: `member-${member.user_id}`,
        identity: member.user_id,
        display_name: member.display_name,
        username: member.username,
        avatar_url: member.avatar_url,
        presence: member.presence || { status: 'offline', custom_text: '' },
        role: member.role || 'MEMBER',
        permissions: member.permissions || [],
        isCurrentUser: member.isCurrentUser,
        isVoiceParticipant: false,
        isMuted: false,
        isScreenSharing: false,
        isCameraOn: false,
        isLocal: false
      }));

    this.participants = [...liveParticipants, ...silentMembers]
      .sort((left, right) => this.participantWeight(right) - this.participantWeight(left));
  }

  private sortMembers(members: any[]): any[] {
    return [...members].sort((left, right) => this.memberWeight(right) - this.memberWeight(left));
  }

  private memberWeight(member: any): number {
    const presence = member?.presence?.status || 'offline';
    return this.roleWeight(member?.role) * 100 + this.presenceWeight(presence) * 10 + (member?.isCurrentUser ? 1 : 0);
  }

  private participantWeight(participant: any): number {
    return (
      (participant.isVoiceParticipant === false ? 0 : 2000) +
      (participant.isScreenSharing ? 1000 : 0) +
      (this.isSpeaking(participant.sid) ? 500 : 0) +
      (participant.isCameraOn ? 120 : 0) +
      (this.roleWeight(participant.role) * 10) +
      (participant.isCurrentUser ? 1 : 0)
    );
  }

  private roleWeight(role: string): number {
    if (role === 'OWNER') return 3;
    if (role === 'MODERATOR') return 2;
    return 1;
  }

  private presenceWeight(status: string): number {
    if (status === 'online') return 4;
    if (status === 'busy') return 3;
    if (status === 'away') return 2;
    return 1;
  }

  private getMemberByIdentity(identity: string): any {
    return this.members.find(member => member.user_id === identity || member.id === identity);
  }

  private isElementExpected(participant: any, source: string): boolean {
    if (!this.isConnected) {
      return false;
    }
    
    const hasScreen = this.hasScreenshare();
    const screensharePart = this.screenshareParticipant;
    
    if (hasScreen && screensharePart) {
      if (source === 'screen_share') {
        return participant.sid === screensharePart.sid;
      } else if (source === 'camera') {
        return !!participant.isCameraOn;
      }
    } else {
      if (source === 'screen_share') {
        return !!participant.isScreenSharing;
      } else if (source === 'camera') {
        return !!participant.isCameraOn;
      }
    }
    return false;
  }

  private attachVideoTracks(): void {
    if (!this.videoElements) return;

    const elements = this.videoElements.toArray();
    const elMap = new Map<string, HTMLVideoElement>();
    elements.forEach(elRef => {
      const el = elRef.nativeElement;
      if (el.id) {
        elMap.set(el.id, el);
      }
    });

    const activeElementIds = new Set<string>();

    this.participants.forEach(participant => {
      if (!participant.raw || typeof participant.raw.getTrackPublications !== 'function') {
        return;
      }

      const publications = participant.raw.getTrackPublications();
      publications.forEach((publication: any) => {
        const track = publication.videoTrack || publication.track;
        if (!track || track.kind !== 'video') {
          return;
        }

        const isScreenShare = publication.source === 'screen_share';
        const targetId = isScreenShare ? 'video-screen-' : 'video-camera-';
        const elementId = targetId + participant.sid;

        if (!this.isElementExpected(participant, publication.source)) {
          // Detach track if it's no longer expected
          const videoEl = elMap.get(elementId);
          if (videoEl && videoEl.dataset['trackSid'] === track.sid) {
            try {
              track.detach(videoEl);
            } catch (e) {
              console.warn('Failed to detach track:', e);
            }
            videoEl.srcObject = null;
            delete videoEl.dataset['trackSid'];
          }
          return;
        }

        const videoEl = elMap.get(elementId);
        if (videoEl) {
          activeElementIds.add(elementId);
          videoEl.muted = true;

          if (videoEl.dataset['trackSid'] !== track.sid || !videoEl.srcObject) {
            try {
              track.attach(videoEl);
              videoEl.dataset['trackSid'] = track.sid;
            } catch (e) {
              console.error('Failed to attach track:', e);
            }
          }
        }
      });
    });

    // Detach elements that are in the DOM but weren't active in our loop
    elMap.forEach((videoEl, elementId) => {
      if (!activeElementIds.has(elementId)) {
        const trackSid = videoEl.dataset['trackSid'];
        if (trackSid) {
          let trackFound = false;
          this.participants.forEach(p => {
            if (p.raw && typeof p.raw.getTrackPublications === 'function') {
              p.raw.getTrackPublications().forEach((pub: any) => {
                const track = pub.videoTrack || pub.track;
                if (track && track.sid === trackSid) {
                  try {
                    track.detach(videoEl);
                  } catch (e) {}
                  trackFound = true;
                }
              });
            }
          });
          videoEl.srcObject = null;
          delete videoEl.dataset['trackSid'];
        }
      }
    });
  }

  public toggleFullscreen(): void {
    const container = document.querySelector('.screenshare-viewport-shell');
    if (container) {
      if (container.requestFullscreen) {
        container.requestFullscreen();
      } else if ((container as any).webkitRequestFullscreen) { /* Safari */
        (container as any).webkitRequestFullscreen();
      } else if ((container as any).msRequestFullscreen) { /* IE11 */
        (container as any).msRequestFullscreen();
      }
    }
  }
}
