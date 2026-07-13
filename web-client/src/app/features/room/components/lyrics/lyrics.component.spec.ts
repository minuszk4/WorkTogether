import { TestBed, ComponentFixture } from '@angular/core/testing';
import { LyricsComponent } from './lyrics.component';
import { LyricsService } from '../../../../core/services/lyrics.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { BehaviorSubject, of } from 'rxjs';

describe('LyricsComponent', () => {
  let component: LyricsComponent;
  let fixture: ComponentFixture<LyricsComponent>;
  
  let mockLyricsService: any;
  let mockPlayerEngine: any;
  let mockStateService: any;
  let mockToastService: any;

  let currentTrackSubject: BehaviorSubject<any>;
  let progressMsSubject: BehaviorSubject<number>;

  beforeEach(async () => {
    currentTrackSubject = new BehaviorSubject<any>(null);
    progressMsSubject = new BehaviorSubject<number>(0);

    mockLyricsService = {
      getLyrics: jasmine.createSpy('getLyrics').and.returnValue(of({
        success: true,
        data: { content: '[00:10.00]Line 1\n[00:20.00]Line 2' }
      })),
      saveLyrics: jasmine.createSpy('saveLyrics').and.returnValue(of({
        success: true
      }))
    };

    mockPlayerEngine = {
      currentTrack$: currentTrackSubject,
      progressMs$: progressMsSubject,
      seekToMs: jasmine.createSpy('seekToMs'),
      currentTrack: null
    };

    mockStateService = {};

    mockToastService = {
      success: jasmine.createSpy('success'),
      error: jasmine.createSpy('error')
    };

    await TestBed.configureTestingModule({
      imports: [LyricsComponent],
      providers: [
        { provide: LyricsService, useValue: mockLyricsService },
        { provide: PlayerEngineService, useValue: mockPlayerEngine },
        { provide: StateService, useValue: mockStateService },
        { provide: ToastService, useValue: mockToastService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(LyricsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile', () => {
    expect(component).toBeTruthy();
  });

  it('should load and parse lyrics when track changes', () => {
    currentTrackSubject.next({ id: 'track-1', title: 'Song 1' });
    fixture.detectChanges();

    expect(mockLyricsService.getLyrics).toHaveBeenCalledWith('track-1');
    expect(component.lyricsLines.length).toBe(2);
    expect(component.lyricsLines[0].text).toBe('Line 1');
    expect(component.lyricsLines[0].timeMs).toBe(10000);
    expect(component.lyricsLines[1].text).toBe('Line 2');
    expect(component.lyricsLines[1].timeMs).toBe(20000);
  });

  it('should update active index based on playback progress', () => {
    currentTrackSubject.next({ id: 'track-1', title: 'Song 1' });
    fixture.detectChanges();

    progressMsSubject.next(5000);
    expect(component.activeIndex).toBe(-1);

    progressMsSubject.next(12000);
    expect(component.activeIndex).toBe(0);

    progressMsSubject.next(25000);
    expect(component.activeIndex).toBe(1);
  });

  it('should save lyrics when submitted', () => {
    component.trackId = 'track-1';
    component.lyricsContent = '[00:05.00]Hello';
    component.saveLyrics();

    expect(mockLyricsService.saveLyrics).toHaveBeenCalledWith('track-1', '[00:05.00]Hello');
    expect(mockToastService.success).toHaveBeenCalled();
  });

  it('allows lyric editing only for an admin access token', () => {
    mockStateService.accessToken = `${btoa(JSON.stringify({ alg: 'none' }))}.${btoa(JSON.stringify({ is_admin: false }))}.signature`;
    expect(component.canEditLyrics).toBeFalse();

    mockStateService.accessToken = `${btoa(JSON.stringify({ alg: 'none' }))}.${btoa(JSON.stringify({ is_admin: true }))}.signature`;
    expect(component.canEditLyrics).toBeTrue();
  });
});
