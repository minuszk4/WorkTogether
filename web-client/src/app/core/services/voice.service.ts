import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
import { ConnectionQuality, Participant, Room, RoomEvent, Track } from 'livekit-client';
import { environment } from '../../../environments/environment';

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
  public connectionState$ = new BehaviorSubject<'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'degraded'>('disconnected');
  private audioHost: HTMLElement | null = null;
  private captionRecorder: MediaRecorder | null = null;
  private captionTimer: ReturnType<typeof setTimeout> | null = null;
  private captionEnabled = false;
  private captionPaused = false;
  private captionCallback: ((audio: Blob) => void) | null = null;

  public async connect(serverUrl: string, token: string, audioDeviceId = ''): Promise<void> {
    if (this.connected$.value) return;
    if (this.connectInFlight) return this.connectInFlight;

    this.connectInFlight = this.doConnect(serverUrl, token, audioDeviceId).finally(() => {
      this.connectInFlight = null;
    });

    return this.connectInFlight;
  }

  private async doConnect(serverUrl: string, token: string, audioDeviceId: string): Promise<void> {
    try {
      this.disconnect();
	  this.connectionState$.next('connecting');

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
	  this.connectionState$.next('connected');
	  if (audioDeviceId) {
		await this.room.switchActiveDevice('audioinput', audioDeviceId);
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
		  this.getAudioHost()?.appendChild(element);
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
	  .on(RoomEvent.Reconnecting, () => this.connectionState$.next('reconnecting'))
	  .on(RoomEvent.Reconnected, () => this.connectionState$.next('connected'))
	  .on(RoomEvent.ConnectionQualityChanged, (quality, participant) => {
		if (participant.isLocal) {
		  this.connectionState$.next(
			quality === ConnectionQuality.Poor || quality === ConnectionQuality.Lost ? 'degraded' : 'connected'
		  );
		}
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
    if (this.captionEnabled) {
      this.captionPaused = muted;
      if (muted) this.stopCaptions();
      else this.startCaptions();
    }
    this.updateParticipants();
  }

  public setCaptionsEnabled(enabled: boolean, onAudio?: (audio: Blob) => void): boolean {
    this.captionEnabled = enabled;
    this.captionPaused = false;
    this.captionCallback = enabled ? onAudio || null : null;
    this.stopCaptions();
    return !enabled || this.startCaptions();
  }

  private startCaptions(): boolean {
    if (!this.captionEnabled || this.captionRecorder || !this.room?.localParticipant || typeof MediaRecorder === 'undefined') return false;
    const track = this.room.localParticipant.getTrackPublication(Track.Source.Microphone)?.audioTrack?.mediaStreamTrack;
    if (!track || track.readyState !== 'live') return false;
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus') ? 'audio/webm;codecs=opus' : 'audio/webm';
    const chunks: Blob[] = [];
    let recorder: MediaRecorder;
    try {
      recorder = new MediaRecorder(new MediaStream([track]), { mimeType });
    } catch {
      return false;
    }
    recorder.ondataavailable = event => {
      if (event.data.size > 0) chunks.push(event.data);
    };
    recorder.onstop = () => {
      if (this.captionRecorder === recorder) this.captionRecorder = null;
      if (this.captionTimer) clearTimeout(this.captionTimer);
      this.captionTimer = null;
      const audio = new Blob(chunks, { type: mimeType });
      if (!this.captionPaused && audio.size > 0) this.captionCallback?.(audio);
      if (this.captionEnabled && !this.captionPaused) this.startCaptions();
    };
    recorder.start();
    this.captionRecorder = recorder;
    this.captionTimer = setTimeout(() => recorder.stop(), 2000);
    return true;
  }

  private stopCaptions(): void {
    if (this.captionTimer) clearTimeout(this.captionTimer);
    this.captionTimer = null;
    if (this.captionRecorder && this.captionRecorder.state !== 'inactive') this.captionRecorder.stop();
    this.captionRecorder = null;
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

  private getAudioHost(): HTMLElement | null {
	if (typeof document === 'undefined') return null;
	if (!this.audioHost) {
	  this.audioHost = document.createElement('div');
	  this.audioHost.hidden = true;
	  this.audioHost.dataset['livekitAudio'] = 'true';
	  document.body.appendChild(this.audioHost);
	}
	return this.audioHost;
  }

  private normalizeLiveKitUrl(serverUrl: string): string {
    let normalizedUrl = serverUrl || environment.livekitUrl;
    normalizedUrl = normalizedUrl.replace(/^http:\/\//i, 'ws://').replace(/^https:\/\//i, 'wss://');

    try {
      const parsed = new URL(normalizedUrl);
      if (parsed.hostname === 'livekit' || parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1') {
        parsed.hostname = window.location.hostname || 'localhost';
      }
      return parsed.toString();
    } catch {
      return environment.livekitUrl;
    }
  }

  private resetState(): void {
	this.stopCaptions();
	this.captionEnabled = false;
	this.captionPaused = false;
	this.captionCallback = null;
	this.audioHost?.remove();
	this.audioHost = null;
    this.room = null;
    this.connected$.next(false);
    this.participants$.next([]);
    this.activeSpeakers$.next([]);
    this.isMuted$.next(false);
    this.isScreenSharing$.next(false);
    this.isCameraActive$.next(false);
	this.connectionState$.next('disconnected');
  }
}
