import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { ApiService } from '../../../../core/services/api.service';
import { StateService } from '../../../../core/services/state.service';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { LyricsComponent } from '../lyrics/lyrics.component';
import { BookmarksComponent } from '../bookmarks/bookmarks.component';

export interface QueueTrack {
  id: string;
  track_id: string;
  title: string;
  artist: string;
  thumbnail_url: string;
  duration_ms: number;
  source_url: string;
  source?: string;
  votes: number;
}

@Component({
  selector: 'app-room-queue',
  standalone: true,
  imports: [CommonModule, FormsModule, LyricsComponent, BookmarksComponent],
  templateUrl: './queue.component.html',
  styleUrl: './queue.component.css'
})
export class QueueComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  public state = inject(StateService);
  private playbackWs = inject(PlaybackWsService);
  private toast = inject(ToastService);

  public activeTab: 'queue' | 'members' | 'lyrics' | 'bookmarks' = 'queue';
  public tracks: QueueTrack[] = [];
  public members: any[] = [];
  public newTrackUrl = '';
  public activePlaylistId = '';
  public isExtracting = false;
  public searchResults: any[] = [];

  private roomId = '';
  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.state.activeRoom$.subscribe(room => {
        if (room?.id) {
          this.roomId = room.id;
          this.setupPlaylist();
        }
      }),
      this.state.activeRoomMembers$.subscribe(members => {
        this.members = members || [];
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(sub => sub.unsubscribe());
  }

  public switchTab(tab: 'queue' | 'members' | 'lyrics' | 'bookmarks'): void {
    this.activeTab = tab;
    if (tab === 'queue') {
      this.loadQueueTracks();
    }
  }

  public get canManageQueue(): boolean {
    return this.state.roomPermissions.includes('CAN_CONTROL_PLAYBACK') || this.state.roomPermissions.includes('CAN_MANAGE_PLAYLIST');
  }

  public get canModerateMembers(): boolean {
    return this.state.roomPermissions.includes('CAN_MODERATE_MEMBERS');
  }

  private async setupPlaylist(): Promise<void> {
    if (!this.roomId) return;

    try {
      this.api.playlist.getRoomPlaylists(this.roomId).subscribe({
        next: (playlists) => {
          if (!playlists || playlists.length === 0) {
            this.api.playlist.create('Default Playlist', this.roomId).subscribe({
              next: (newPlaylist) => {
                this.activePlaylistId = newPlaylist.id;
                this.loadQueueTracks();
              },
              error: (err) => console.error('Error creating playlist:', err)
            });
          } else {
            this.activePlaylistId = playlists[0].id;
            this.loadQueueTracks();
          }
        },
        error: (err) => console.error('Error fetching room playlists:', err)
      });
    } catch (err) {
      console.error('Error setting up playlist:', err);
    }
  }

  public loadQueueTracks(): void {
    if (!this.activePlaylistId) return;

    this.api.playlist.getTracks(this.activePlaylistId).subscribe({
      next: (tracks) => {
        this.tracks = tracks || [];
      },
      error: (err) => {
        console.error('Error loading tracks:', err);
        this.toast.error('Khong the tai danh sach hang doi.');
      }
    });
  }

  public onAddTrackSubmit(event: Event): void {
    event.preventDefault();
    const url = this.newTrackUrl.trim();
    if (!url || !this.activePlaylistId) return;

    if (url.startsWith('http://') || url.startsWith('https://')) {
      this.isExtracting = true;
      this.toast.info('Dang trich xuat video tu YouTube...');

      this.api.music.extract(url).subscribe({
        next: (track) => {
          this.toast.info('Them bai hat vao danh sach phat...');
          this.api.playlist.addTrack(this.activePlaylistId, track).subscribe({
            next: () => {
              this.toast.success(`Da them bai hat "${track.title}" vao hang doi.`);
              this.newTrackUrl = '';
              this.isExtracting = false;
              this.loadQueueTracks();
            },
            error: (err) => {
              this.toast.error('Khong the them bai hat: ' + err.message);
              this.isExtracting = false;
            }
          });
        },
        error: (err) => {
          this.toast.error('Trich xuat link YouTube that bai: ' + err.message);
          this.isExtracting = false;
        }
      });
      return;
    }

    this.searchYoutube(url);
  }

  public searchYoutube(keyword: string): void {
    this.isExtracting = true;
    this.toast.info('Dang tim kiem tren YouTube...');
    this.api.music.search(keyword).subscribe({
      next: (results) => {
        this.searchResults = results || [];
        this.isExtracting = false;
        if (this.searchResults.length === 0) {
          this.toast.info('Khong tim thay ket qua nao.');
        }
      },
      error: (err) => {
        this.toast.error('Tim kiem that bai: ' + err.message);
        this.isExtracting = false;
      }
    });
  }

  public onAddSearchResult(result: any): void {
    if (!this.activePlaylistId) return;

    this.isExtracting = true;
    this.toast.info('Dang them bai hat vao hang doi...');

    this.api.playlist.addTrack(this.activePlaylistId, result).subscribe({
      next: () => {
        this.toast.success(`Da them bai hat "${result.title}" vao hang doi.`);
        this.clearSearch();
        this.newTrackUrl = '';
        this.isExtracting = false;
        this.loadQueueTracks();
      },
      error: (err) => {
        this.toast.error('Khong the them bai hat: ' + err.message);
        this.isExtracting = false;
      }
    });
  }

  public clearSearch(): void {
    this.searchResults = [];
  }

  public onVoteClick(trackId: string, event: MouseEvent): void {
    event.stopPropagation();
    this.api.playlist.voteTrack(trackId, 'up').subscribe({
      next: () => {
        this.loadQueueTracks();
      },
      error: () => {
        this.toast.error('Binh chon that bai.');
      }
    });
  }

  public onRemoveClick(trackId: string, event: MouseEvent): void {
    event.stopPropagation();
    if (!this.activePlaylistId) return;

    this.api.playlist.removeTrack(this.activePlaylistId, trackId).subscribe({
      next: () => {
        this.toast.success('Da xoa khoi hang doi.');
        this.loadQueueTracks();
      },
      error: () => {
        this.toast.error('Xoa that bai.');
      }
    });
  }

  public onTrackClick(track: QueueTrack): void {
    const newTrack = {
      id: track.track_id || track.id,
      title: track.title,
      artist: track.artist,
      thumbnail_url: track.thumbnail_url,
      duration_ms: track.duration_ms,
      source_url: track.source_url,
      source: track.source
    };

    this.playbackWs.playbackSync$.next({
      state: 'playing',
      current_track: newTrack,
      position_ms: 0,
      updated_at: Date.now()
    });

    this.playbackWs.sendControlCommand('play', 0);
  }

  public canKick(member: any): boolean {
    return this.canModerateMembers && !member.isCurrentUser && member.role !== 'OWNER';
  }

  public onKickClick(member: any, event: MouseEvent): void {
    event.stopPropagation();
    if (!this.roomId || !this.canKick(member)) return;

    this.api.room.kick(this.roomId, member.user_id).subscribe({
      next: () => {
        this.toast.success(`Da kick ${member.display_name}.`);
        this.members = this.members.filter(item => item.user_id !== member.user_id);
        this.state.activeRoomMembers$.next(this.members);
      },
      error: (err) => {
        this.toast.error('Kick that bai: ' + err.message);
      }
    });
  }

  public getInitials(member: any): string {
    const label = member?.display_name || member?.username || 'WT';
    return label.slice(0, 2).toUpperCase();
  }

  public getStatusText(member: any): string {
    return member?.presence?.custom_text || member?.presence?.status || 'offline';
  }

  public dragIndex: number | null = null;

  public onDragStart(index: number, event: DragEvent): void {
    if (!this.canManageQueue) {
      event.preventDefault();
      return;
    }

    this.dragIndex = index;
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData('text/plain', index.toString());
    }
  }

  public onDragOver(_index: number, event: DragEvent): void {
    if (!this.canManageQueue || this.dragIndex === null) return;
    event.preventDefault();
  }

  public onDrop(index: number, event: DragEvent): void {
    if (!this.canManageQueue || this.dragIndex === null || this.dragIndex === index) {
      return;
    }
    event.preventDefault();

    const movedTrack = this.tracks[this.dragIndex];
    this.api.playlist.moveTrack(this.activePlaylistId, movedTrack.id, index).subscribe({
      next: () => {
        this.toast.success('Da sap xep lai hang doi.');
        this.loadQueueTracks();
      },
      error: (err) => {
        this.toast.error('Khong the sap xep lai: ' + err.message);
        this.loadQueueTracks();
      }
    });

    this.dragIndex = null;
  }

  public onDragEnd(): void {
    this.dragIndex = null;
  }
}
