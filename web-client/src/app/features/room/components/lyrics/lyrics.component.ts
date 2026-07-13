import { Component, OnDestroy, OnInit, inject, ElementRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { LyricsService } from '../../../../core/services/lyrics.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';

interface LyricLine {
  timeMs: number;
  text: string;
}

@Component({
  selector: 'app-room-lyrics',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './lyrics.component.html',
  styleUrl: './lyrics.component.css'
})
export class LyricsComponent implements OnInit, OnDestroy {
  private lyricsService = inject(LyricsService);
  private engine = inject(PlayerEngineService);
  public state = inject(StateService);
  private toast = inject(ToastService);

  @ViewChild('linesContainer') linesContainer!: ElementRef<HTMLDivElement>;

  public trackId: string | null = null;
  public trackTitle = '';
  public lyricsLines: LyricLine[] = [];
  public activeIndex = -1;
  public isEditing = false;
  public lyricsContent = '';
  public isSaving = false;
  public isLoading = false;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.engine.currentTrack$.subscribe(track => {
        if (track && track.id) {
          if (this.trackId !== track.id) {
            this.trackId = track.id;
            this.trackTitle = track.title;
            this.loadLyrics(track.id);
          }
        } else {
          this.trackId = null;
          this.trackTitle = '';
          this.lyricsLines = [];
          this.lyricsContent = '';
          this.activeIndex = -1;
        }
      }),
      this.engine.progressMs$.subscribe(ms => {
        this.updateActiveLine(ms);
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  public get canEditLyrics(): boolean {
    const token = this.state.accessToken;
    if (!token) return false;
    try {
      const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
      return payload.is_admin === true;
    } catch {
      return false;
    }
  }

  private loadLyrics(trackId: string): void {
    this.isLoading = true;
    this.lyricsService.getLyrics(trackId).subscribe({
      next: (res) => {
        this.isLoading = false;
        if (res && res.success && res.data) {
          this.lyricsContent = res.data.content || '';
          this.parseLRC(this.lyricsContent);
        } else {
          this.lyricsLines = [];
          this.lyricsContent = '';
        }
      },
      error: () => {
        this.isLoading = false;
        this.lyricsLines = [];
        this.lyricsContent = '';
      }
    });
  }

  private parseLRC(lrcText: string): void {
    const lines = lrcText.split('\n');
    const result: LyricLine[] = [];
    const regex = /\[(\d+):(\d+)(?:[.:](\d+))?\](.*)/;

    for (const line of lines) {
      const match = line.match(regex);
      if (match) {
        const min = parseInt(match[1], 10);
        const sec = parseInt(match[2], 10);
        let ms = 0;
        if (match[3]) {
          const msStr = match[3];
          if (msStr.length === 2) {
            ms = parseInt(msStr, 10) * 10;
          } else if (msStr.length === 3) {
            ms = parseInt(msStr, 10);
          }
        }
        const timeMs = min * 60 * 1000 + sec * 1000 + ms;
        const text = match[4].trim();
        result.push({ timeMs, text });
      }
    }

    this.lyricsLines = result.sort((a, b) => a.timeMs - b.timeMs);
    this.activeIndex = -1;
  }

  private updateActiveLine(currentMs: number): void {
    if (this.lyricsLines.length === 0) return;

    let index = -1;
    for (let i = 0; i < this.lyricsLines.length; i++) {
      if (currentMs >= this.lyricsLines[i].timeMs) {
        index = i;
      } else {
        break;
      }
    }

    if (index !== this.activeIndex) {
      this.activeIndex = index;
      this.scrollToActiveLine();
    }
  }

  private scrollToActiveLine(): void {
    if (!this.linesContainer) return;
    setTimeout(() => {
      const container = this.linesContainer.nativeElement;
      const activeEl = container.querySelector('.lyric-line.active') as HTMLElement;
      if (activeEl) {
        const containerHeight = container.clientHeight;
        const activeHeight = activeEl.clientHeight;
        const activeTop = activeEl.offsetTop;
        container.scrollTo({
          top: activeTop - containerHeight / 2 + activeHeight / 2,
          behavior: 'smooth'
        });
      }
    }, 50);
  }

  public openEditModal(): void {
	if (!this.canEditLyrics) return;
    this.isEditing = true;
  }

  public closeEditModal(): void {
    this.isEditing = false;
  }

  public saveLyrics(): void {
    if (!this.trackId) return;

    this.isSaving = true;
    this.lyricsService.saveLyrics(this.trackId, this.lyricsContent).subscribe({
      next: (res) => {
        this.isSaving = false;
        if (res && res.success) {
          this.toast.success('Lưu lời bài hát thành công.');
          this.parseLRC(this.lyricsContent);
          this.isEditing = false;
        } else {
          this.toast.error('Không thể lưu lời bài hát.');
        }
      },
      error: (err) => {
        this.isSaving = false;
        this.toast.error('Lỗi khi lưu lời bài hát: ' + (err.message || err));
      }
    });
  }

  public seekTo(timeMs: number): void {
    this.engine.seekToMs(timeMs);
  }
}
