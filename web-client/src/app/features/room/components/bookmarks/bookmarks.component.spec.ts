import { TestBed, ComponentFixture } from '@angular/core/testing';
import { BookmarksComponent } from './bookmarks.component';
import { BookmarksService } from '../../../../core/services/bookmarks.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { BehaviorSubject, of } from 'rxjs';

describe('BookmarksComponent', () => {
  let component: BookmarksComponent;
  let fixture: ComponentFixture<BookmarksComponent>;

  let mockBookmarksService: any;
  let mockPlayerEngine: any;
  let mockStateService: any;
  let mockToastService: any;

  let activeRoomSubject: BehaviorSubject<any>;

  beforeEach(async () => {
    activeRoomSubject = new BehaviorSubject<any>({ id: 'room-1', name: 'Room 1' });

    mockBookmarksService = {
      getBookmarks: jasmine.createSpy('getBookmarks').and.returnValue(of({
        success: true,
        data: [
          { id: 'b1', note: 'Drop', position_ms: 15000, created_at: '' }
        ]
      })),
      saveBookmark: jasmine.createSpy('saveBookmark').and.returnValue(of({
        success: true
      })),
      deleteBookmark: jasmine.createSpy('deleteBookmark').and.returnValue(of({
        success: true
      }))
    };

    mockPlayerEngine = {
      currentTrack: { id: 'track-1', title: 'Song 1' },
      getLocalProgress: jasmine.createSpy('getLocalProgress').and.returnValue(12000),
      seekToMs: jasmine.createSpy('seekToMs'),
      formatTime: jasmine.createSpy('formatTime').and.returnValue('00:12')
    };

    mockStateService = {
      activeRoom$: activeRoomSubject,
      roomPermissions: ['CAN_CONTROL_PLAYBACK']
    };

    mockToastService = {
      success: jasmine.createSpy('success'),
      error: jasmine.createSpy('error'),
      info: jasmine.createSpy('info')
    };

    await TestBed.configureTestingModule({
      imports: [BookmarksComponent],
      providers: [
        { provide: BookmarksService, useValue: mockBookmarksService },
        { provide: PlayerEngineService, useValue: mockPlayerEngine },
        { provide: StateService, useValue: mockStateService },
        { provide: ToastService, useValue: mockToastService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(BookmarksComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and load bookmarks', () => {
    expect(component).toBeTruthy();
    expect(mockBookmarksService.getBookmarks).toHaveBeenCalledWith('room-1');
    expect(component.bookmarks.length).toBe(1);
    expect(component.bookmarks[0].note).toBe('Drop');
  });

  it('should save bookmark when added', () => {
    component.note = 'Awesome part';
    component.addBookmark();

    expect(mockBookmarksService.saveBookmark).toHaveBeenCalledWith('room-1', 'track-1', 12000, 'Awesome part');
    expect(mockToastService.success).toHaveBeenCalled();
  });

  it('should seek to bookmark position', () => {
    component.seekToBookmark(15000);

    expect(mockPlayerEngine.seekToMs).toHaveBeenCalledWith(15000);
  });

  it('should delete bookmark', () => {
    const dummyEvent = new MouseEvent('click');
    component.deleteBookmark('b1', dummyEvent);

    expect(mockBookmarksService.deleteBookmark).toHaveBeenCalledWith('room-1', 'b1');
    expect(mockToastService.success).toHaveBeenCalled();
  });
});
