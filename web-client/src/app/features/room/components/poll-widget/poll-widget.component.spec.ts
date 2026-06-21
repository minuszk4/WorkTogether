import { TestBed, ComponentFixture } from '@angular/core/testing';
import { PollWidgetComponent } from './poll-widget.component';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';
import { BehaviorSubject, Subject } from 'rxjs';

describe('PollWidgetComponent', () => {
  let component: PollWidgetComponent;
  let fixture: ComponentFixture<PollWidgetComponent>;

  let mockPlaybackWsService: any;
  let pollSubject: BehaviorSubject<any>;
  let pollVotesSubject: BehaviorSubject<any>;
  let pollEndSubject: Subject<any>;

  beforeEach(async () => {
    pollSubject = new BehaviorSubject<any>(null);
    pollVotesSubject = new BehaviorSubject<any>({});
    pollEndSubject = new Subject<any>();

    mockPlaybackWsService = {
      poll$: pollSubject,
      pollVotes$: pollVotesSubject,
      pollEnd$: pollEndSubject,
      voteForTrack: jasmine.createSpy('voteForTrack')
    };

    await TestBed.configureTestingModule({
      imports: [PollWidgetComponent],
      providers: [
        { provide: PlaybackWsService, useValue: mockPlaybackWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PollWidgetComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and be hidden by default', () => {
    expect(component).toBeTruthy();
    expect(component.showWidget).toBeFalse();
  });

  it('should show widget when poll starts', () => {
    pollSubject.next({
      candidates: [
        { id: 'c1', title: 'Song 1', artist: 'Artist 1' },
        { id: 'c2', title: 'Song 2', artist: 'Artist 2' }
      ],
      duration: 30000
    });
    fixture.detectChanges();

    expect(component.showWidget).toBeTrue();
    expect(component.candidates.length).toBe(2);
  });

  it('should register vote and disable subsequent voting', () => {
    pollSubject.next({
      candidates: [{ id: 'c1', title: 'Song 1' }],
      duration: 30000
    });
    fixture.detectChanges();

    component.vote('c1');
    expect(mockPlaybackWsService.voteForTrack).toHaveBeenCalledWith('c1');
    expect(component.votedTrackId).toBe('c1');

    // Voting again should do nothing
    component.vote('c2');
    expect(mockPlaybackWsService.voteForTrack).toHaveBeenCalledTimes(1);
  });

  it('should calculate vote percentage correctly', () => {
    pollSubject.next({
      candidates: [{ id: 'c1' }, { id: 'c2' }],
      duration: 30000
    });
    pollVotesSubject.next({
      c1: 3,
      c2: 1
    });
    fixture.detectChanges();

    expect(component.getTotalVotes()).toBe(4);
    expect(component.getVotePct('c1')).toBe(75);
    expect(component.getVotePct('c2')).toBe(25);
  });

  it('should show winner toast when poll ends', () => {
    pollEndSubject.next({
      winner: { title: 'Winning Song' }
    });
    fixture.detectChanges();

    expect(component.showWinner).toBeTrue();
    expect(component.winnerName).toBe('Winning Song');
  });
});
