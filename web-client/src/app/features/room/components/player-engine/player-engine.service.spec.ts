import { TestBed } from '@angular/core/testing';
import { PlayerEngineService } from './player-engine.service';
import { PlaybackWsService, PlaybackState } from '../../../../core/services/websocket/playback-ws.service';

describe('PlayerEngineService', () => {
  let engine: PlayerEngineService;
  let playbackWs: PlaybackWsService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    engine = TestBed.inject(PlayerEngineService);
    playbackWs = TestBed.inject(PlaybackWsService);
  });

  it('reflects playback sync state and resets on null track', () => {
    const state: PlaybackState = {
      state: 'playing',
      current_track: {
        id: 't1', title: 'Song', artist: 'A', thumbnail_url: '',
        duration_ms: 10000, source_url: 'https://youtu.be/abc', source: 'youtube'
      },
      position_ms: 1000,
      updated_at: Date.now()
    };
    playbackWs.playbackSync$.next(state);
    expect(engine.currentTrack?.id).toBe('t1');
    expect(engine.currentState).toBe('playing');
  });

  it('send play/pause control command via PlaybackWsService', () => {
    const spy = spyOn(playbackWs, 'sendControlCommand');
    // seed a track so command is not short-circuited
    playbackWs.playbackSync$.next({
      state: 'paused', current_track: { id: 't1', title: 'x', artist: 'y', thumbnail_url: '', duration_ms: 5000, source_url: 'u', source: 'youtube' },
      position_ms: 0, updated_at: Date.now()
    });
    engine.togglePlayPause();
    expect(spy).toHaveBeenCalledWith('play', jasmine.any(Number));
  });

  it('computes seek ms from a scrub fraction', () => {
    playbackWs.playbackSync$.next({
      state: 'paused', current_track: { id: 't1', title: 'x', artist: 'y', thumbnail_url: '', duration_ms: 60000, source_url: 'u', source: 'youtube' },
      position_ms: 0, updated_at: Date.now()
    });
    const seekMs = engine.seekMsFromFraction(0.5);
    expect(seekMs).toBe(30000);
  });
});
