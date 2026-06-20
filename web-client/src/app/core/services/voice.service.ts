import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
import { Participant, Room, RoomEvent } from 'livekit-client';

@Injectable({
  providedIn: 'root'
})
export class VoiceService {
  private room: Room | null = null;
  private connectInFlight: Promise<void> | null = null;

  public connected$ = new BehaviorSubject<boolean>(false);
  public participants$ = new BehaviorSubject<any[]>([]);
  public activeSpeakers$ = new BehaviorSubject<string[]>([]);
  public isMuted$ = new BehaviorSubject<boolean>(false);
  public isScreenSharing$ = new BehaviorSubject<boolean>(false);
  public isCameraActive$ = new BehaviorSubject<boolean>(false);

  constructor() {
    this.forceWebRtcTcpOnly();
  }

  private forceWebRtcTcpOnly(): void {
    if (typeof window === 'undefined' || (window as any).__webrtc_forced_tcp) {
      return;
    }
    (window as any).__webrtc_forced_tcp = true;

    // Helper to strip UDP candidates from SDP text
    const stripUdpFromSdp = (sdp: string): string => {
      const lines = sdp.split('\n');
      const filteredLines = lines.filter(line => {
        if (line.startsWith('a=candidate:') && line.toLowerCase().includes(' udp ')) {
          return false;
        }
        return true;
      });
      return filteredLines.join('\n');
    };

    // 1. Filter UDP candidates added via trickle ICE
    const originalAddIceCandidate = RTCPeerConnection.prototype.addIceCandidate;
    RTCPeerConnection.prototype.addIceCandidate = function (
      candidate?: any,
      successCallback?: any,
      failureCallback?: any
    ): Promise<void> {
      if (candidate) {
        const candidateStr = typeof candidate === 'string' ? candidate : (candidate.candidate || '');
        if (candidateStr.toLowerCase().includes(' udp ')) {
          if (successCallback) successCallback();
          return Promise.resolve();
        }
      }
      return originalAddIceCandidate.apply(this, arguments as any);
    };

    // 2. Filter UDP candidates from Remote Session Description (SDP)
    const originalSetRemoteDescription = RTCPeerConnection.prototype.setRemoteDescription;
    RTCPeerConnection.prototype.setRemoteDescription = function (description: RTCSessionDescriptionInit): Promise<void> {
      let desc = description;
      if (description && description.sdp) {
        desc = {
          type: description.type,
          sdp: stripUdpFromSdp(description.sdp)
        };
      }
      return (originalSetRemoteDescription as any).call(this, desc);
    };

    // 3. Filter UDP candidates from Local Session Description (SDP)
    const originalSetLocalDescription = RTCPeerConnection.prototype.setLocalDescription;
    RTCPeerConnection.prototype.setLocalDescription = function (description: RTCSessionDescriptionInit): Promise<void> {
      let desc = description;
      if (description && description.sdp) {
        desc = {
          type: description.type,
          sdp: stripUdpFromSdp(description.sdp)
        };
      }
      return (originalSetLocalDescription as any).call(this, desc);
    };

    console.log('[WebRTC] Forced TCP-only mode enabled. Filtered out UDP candidates.');
  }

  public async connect(serverUrl: string, token: string): Promise<void> {
    if (this.connected$.value) return;
    if (this.connectInFlight) return this.connectInFlight;

    this.connectInFlight = this.doConnect(serverUrl, token).finally(() => {
      this.connectInFlight = null;
    });

    return this.connectInFlight;
  }

  private async doConnect(serverUrl: string, token: string): Promise<void> {
    try {
      this.disconnect();

      this.room = new Room({
        adaptiveStream: true,
        dynacast: true
      });

      this.setupRoomListeners();

      await this.room.connect(this.normalizeLiveKitUrl(serverUrl), token);
      if (!this.room) {
        throw new Error('Kết nối LiveKit bị hủy hoặc không thành công.');
      }
      this.connected$.next(true);

      if (this.room.localParticipant) {
        await this.room.localParticipant.setMicrophoneEnabled(true);
      }
      this.updateParticipants();
      this.syncLocalState();
    } catch (err) {
      console.error('LiveKit connect failed:', err);
      this.resetState();
      throw err;
    }
  }

  private setupRoomListeners(): void {
    if (!this.room) return;

    this.room
      .on(RoomEvent.ParticipantConnected, () => this.updateParticipants())
      .on(RoomEvent.ParticipantDisconnected, () => this.updateParticipants())
      .on(RoomEvent.LocalTrackPublished, () => this.updateParticipants())
      .on(RoomEvent.LocalTrackUnpublished, () => this.updateParticipants())
      .on(RoomEvent.TrackPublished, () => this.updateParticipants())
      .on(RoomEvent.TrackUnpublished, () => this.updateParticipants())
      .on(RoomEvent.TrackMuted, () => this.updateParticipants())
      .on(RoomEvent.TrackUnmuted, () => this.updateParticipants())
      .on(RoomEvent.TrackSubscriptionStatusChanged, () => this.updateParticipants())
      .on(RoomEvent.TrackSubscribed, (track) => {
        if (track.kind === 'audio') {
          const element = track.attach();
          if (typeof document !== 'undefined') {
            document.body.appendChild(element);
          }
        }
        this.updateParticipants();
      })
      .on(RoomEvent.TrackUnsubscribed, (track) => {
        if (track.kind === 'audio') {
          const detached = track.detach();
          if (typeof document !== 'undefined') {
            detached.forEach(el => el.remove());
          }
        }
        this.updateParticipants();
      })
      .on(RoomEvent.ActiveSpeakersChanged, (speakers) => {
        this.activeSpeakers$.next(speakers.map(speaker => speaker.sid));
        this.updateParticipants();
      })
      .on(RoomEvent.Disconnected, () => {
        this.resetState();
      });
  }

  private updateParticipants(): void {
    if (!this.room || !this.room.localParticipant) return;

    const participants = [
      this.toParticipantView(this.room.localParticipant, true),
      ...Array.from(this.room.remoteParticipants.values()).map(participant => this.toParticipantView(participant, false))
    ];

    console.log('[VoiceService Debug] updateParticipants. List:', participants.map(p => ({
      identity: p.identity,
      isLocal: p.isLocal,
      isMuted: p.isMuted,
      isScreenSharing: p.isScreenSharing,
      isCameraOn: p.isCameraOn,
      trackPubs: Array.from((p.raw as any).trackPublications?.values() || []).map((pub: any) => ({
        sid: pub.trackSid,
        source: pub.source,
        kind: pub.kind,
        hasTrack: !!pub.track,
        isMuted: pub.isMuted
      }))
    })));

    this.participants$.next(participants);
    this.syncLocalState();
  }

  private toParticipantView(participant: Participant, isLocal: boolean): any {
    return {
      sid: participant.sid,
      identity: participant.identity,
      isLocal,
      isMuted: !participant.isMicrophoneEnabled || this.isMicrophoneMuted(participant),
      isScreenSharing: this.hasActiveSource(participant, 'screen_share'),
      isCameraOn: this.hasActiveSource(participant, 'camera'),
      raw: participant
    };
  }

  private isMicrophoneMuted(participant: Participant): boolean {
    const publications = this.getTrackPublications(participant);
    const microphonePublication = publications.find(publication => 
      publication.source === 'microphone' || (publication.track && publication.track.source === 'microphone')
    );
    if (!microphonePublication) {
      return true;
    }

    return !!microphonePublication.isMuted || !microphonePublication.track;
  }

  private hasActiveSource(participant: Participant, source: string): boolean {
    return this.getTrackPublications(participant).some(
      publication => (publication.source === source || (publication.track && publication.track.source === source)) && 
                     !publication.isMuted && !!publication.track
    );
  }

  private getTrackPublications(participant: Participant): any[] {
    const publicationMap = (participant as any).trackPublications;
    if (!publicationMap || typeof publicationMap.values !== 'function') {
      return [];
    }

    return Array.from(publicationMap.values());
  }

  private syncLocalState(): void {
    if (!this.room || !this.room.localParticipant) return;

    const localParticipant = this.toParticipantView(this.room.localParticipant, true);
    this.isMuted$.next(localParticipant.isMuted);
    this.isScreenSharing$.next(localParticipant.isScreenSharing);
    this.isCameraActive$.next(localParticipant.isCameraOn);
  }

  public async setMute(muted: boolean): Promise<void> {
    if (!this.room || !this.room.localParticipant) return;

    await this.room.localParticipant.setMicrophoneEnabled(!muted);
    this.updateParticipants();
  }

  public async setScreenShare(enabled: boolean): Promise<void> {
    if (!this.room || !this.room.localParticipant) return;

    if (enabled) {
      await this.room.localParticipant.setScreenShareEnabled(true, {
        resolution: {
          width: 1280,
          height: 720,
          frameRate: 10
        }
      });
    } else {
      await this.room.localParticipant.setScreenShareEnabled(false);
    }
    this.updateParticipants();
  }

  public async setCamera(enabled: boolean): Promise<void> {
    if (!this.room || !this.room.localParticipant) return;

    await this.room.localParticipant.setCameraEnabled(enabled);
    this.updateParticipants();
  }

  public disconnect(): void {
    if (this.room) {
      this.room.disconnect();
    }
    this.resetState();
  }

  private normalizeLiveKitUrl(serverUrl: string): string {
    let normalizedUrl = serverUrl || 'ws://localhost:7880';
    normalizedUrl = normalizedUrl.replace(/^http:\/\//i, 'ws://').replace(/^https:\/\//i, 'wss://');

    try {
      const parsed = new URL(normalizedUrl);
      if (parsed.hostname === 'livekit' || parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1') {
        parsed.hostname = window.location.hostname || 'localhost';
      }
      return parsed.toString();
    } catch {
      return 'ws://localhost:7880';
    }
  }

  private resetState(): void {
    this.room = null;
    this.connected$.next(false);
    this.participants$.next([]);
    this.activeSpeakers$.next([]);
    this.isMuted$.next(false);
    this.isScreenSharing$.next(false);
    this.isCameraActive$.next(false);
  }
}
