import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { BookmarksService } from '../../../../core/services/bookmarks.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';

export interface Bookmark {
  id: string;
  room_id: string;
  user_id: string;
  track_id: string;
  position_ms: number;
  note: string;
  created_at: string;
}

@Component({
  selector: 'app-room-bookmarks',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './bookmarks.component.html',
  styleUrl: './bookmarks.component.css'
})
export class BookmarksComponent implements OnInit, OnDestroy {
  private bookmarksService = inject(BookmarksService);
  public engine = inject(PlayerEngineService);
  public state = inject(StateService);
  private toast = inject(ToastService);

  public roomId = '';
  public bookmarks: Bookmark[] = [];
  public note = '';
  public isSaving = false;
  public isLoading = false;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.state.activeRoom$.subscribe(room => {
        if (room?.id) {
          this.roomId = room.id;
          this.loadBookmarks();
        }
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  public loadBookmarks(): void {
    if (!this.roomId) return;
    this.isLoading = true;
    this.bookmarksService.getBookmarks(this.roomId).subscribe({
      next: (res) => {
        this.isLoading = false;
        if (res && res.success) {
          this.bookmarks = res.data || [];
        } else {
          this.bookmarks = [];
        }
      },
      error: (err) => {
        this.isLoading = false;
        console.error('Error loading bookmarks:', err);
      }
    });
  }

  public addBookmark(): void {
    if (!this.roomId || !this.engine.currentTrack) {
      this.toast.error('Không có bài hát nào đang phát.');
      return;
    }

    const trackId = this.engine.currentTrack.id;
    const positionMs = this.engine.getLocalProgress();
    const noteText = this.note.trim() || `Thẻ ghi nhớ lúc ${this.engine.formatTime(positionMs)}`;

    this.isSaving = true;
    this.bookmarksService.saveBookmark(this.roomId, trackId, positionMs, noteText).subscribe({
      next: (res) => {
        this.isSaving = false;
        if (res && res.success) {
          this.toast.success('Đã lưu thẻ ghi nhớ.');
          this.note = '';
          this.loadBookmarks();
        } else {
          this.toast.error('Không thể lưu thẻ ghi nhớ.');
        }
      },
      error: (err) => {
        this.isSaving = false;
        this.toast.error('Lỗi khi lưu thẻ ghi nhớ: ' + (err.message || err));
      }
    });
  }

  public seekToBookmark(positionMs: number): void {
    this.engine.seekToMs(positionMs);
    this.toast.info(`Đang nhảy tới ${this.engine.formatTime(positionMs)}`);
  }

  public deleteBookmark(id: string, event: MouseEvent): void {
    event.stopPropagation();
    if (!this.roomId) return;

    this.bookmarksService.deleteBookmark(this.roomId, id).subscribe({
      next: (res) => {
        if (res && res.success) {
          this.toast.success('Đã xóa thẻ ghi nhớ.');
          this.loadBookmarks();
        } else {
          this.toast.error('Không thể xóa thẻ ghi nhớ.');
        }
      },
      error: (err) => {
        this.toast.error('Lỗi khi xóa thẻ ghi nhớ: ' + (err.message || err));
      }
    });
  }

  public formatTime(ms: number): string {
    return this.engine.formatTime(ms);
  }
}
