import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { ApiService } from '../../../../core/services/api.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { RoomUiStateService } from '../../room-ui-state.service';

@Component({
  selector: 'app-history-stats',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './history-stats.component.html',
  styleUrl: './history-stats.component.css'
})
export class HistoryStatsComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  public state = inject(StateService);
  private toast = inject(ToastService);
  private uiState = inject(RoomUiStateService);

  public roomId = '';
  public activePlaylistId = '';
  public stats: any = null;
  public history: any[] = [];
  public isLoading = false;
  public isQueueing = false;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.state.activeRoom$.subscribe(room => {
        if (room?.id) {
          const changed = this.roomId !== room.id;
          this.roomId = room.id;
          if (changed) {
            this.setupPlaylist();
            this.refreshData();
          }
        }
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  private setupPlaylist(): void {
    if (!this.roomId) return;
    this.api.playlist.getRoomPlaylists(this.roomId).subscribe({
      next: (playlists) => {
        if (playlists && playlists.length > 0) {
          this.activePlaylistId = playlists[0].id;
        }
      },
      error: (err) => console.error('Error fetching playlists in stats:', err)
    });
  }

  public refreshData(): void {
    if (!this.roomId) return;
    this.isLoading = true;

    // Load Stats
    this.api.music.getStats(this.roomId).subscribe({
      next: (data) => {
        this.stats = data;
        this.isLoading = false;
      },
      error: (err) => {
        console.error('Error loading stats:', err);
        this.isLoading = false;
      }
    });

    // Load History
    this.api.music.getHistory(this.roomId).subscribe({
      next: (data) => {
        this.history = data || [];
      },
      error: (err) => {
        console.error('Error loading history:', err);
      }
    });
  }

  public addToQueue(track: any, event: MouseEvent): void {
    event.stopPropagation();
    if (!this.activePlaylistId) {
      this.toast.error('Hàng đợi chưa sẵn sàng.');
      return;
    }

    this.isQueueing = true;
    this.toast.info(`Đang thêm "${track.title}" vào hàng đợi...`);

    const payload = {
      id: track.id,
      title: track.title,
      artist: track.artist || 'Unknown',
      thumbnail_url: track.thumbnail_url || '',
      duration_ms: track.duration_ms || 180000,
      source_url: track.source_url || ''
    };

    this.api.playlist.addTrack(this.activePlaylistId, payload).subscribe({
      next: () => {
        this.toast.success(`Đã thêm "${track.title}" vào hàng đợi.`);
        this.isQueueing = false;
      },
      error: (err) => {
        this.toast.error('Thêm bài hát thất bại: ' + err.message);
        this.isQueueing = false;
      }
    });
  }

  public formatPlayTime(durationMs: number): string {
    if (!durationMs) return '0 phút';
    const totalSecs = Math.floor(durationMs / 1000);
    const hours = Math.floor(totalSecs / 3600);
    const mins = Math.floor((totalSecs % 3600) / 60);

    if (hours > 0) {
      return `${hours} giờ ${mins} phút`;
    }
    return `${mins} phút`;
  }

  public formatDuration(durationMs: number): string {
    if (!durationMs) return '00:00';
    const secs = Math.floor(durationMs / 1000);
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  }

  public formatTimeAgo(timeStr: string): string {
    try {
      const past = new Date(timeStr);
      const diffMs = Date.now() - past.getTime();
      const diffMins = Math.floor(diffMs / 60000);

      if (diffMins < 1) return 'Vừa xong';
      if (diffMins < 60) return `${diffMins} phút trước`;
      
      const diffHours = Math.floor(diffMins / 60);
      if (diffHours < 24) return `${diffHours} giờ trước`;

      return past.toLocaleDateString('vi-VN');
    } catch {
      return 'Không rõ';
    }
  }

  public close(): void {
    this.uiState.toggleStats(false);
  }
}
