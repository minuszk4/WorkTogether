import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { ApiService } from '../../../../core/services/api.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { RoomUiStateService } from '../../room-ui-state.service';

@Component({
  selector: 'app-room-identity',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './room-identity.component.html',
  styleUrl: './room-identity.component.css'
})
export class RoomIdentityComponent implements OnInit, OnDestroy {
  public api = inject(ApiService);
  public state = inject(StateService);
  private toast = inject(ToastService);
  private uiState = inject(RoomUiStateService);

  public settingsName = '';
  public settingsDescription = '';
  public settingsAddMusicPolicy = 'all';
  public settingsAvatarUrl = '';
  public settingsRules = '';
  public settingsTheme = 'cool-ocean';
  public isSavingSettings = false;

  public roomId = '';
  public isOwner = false;
  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.state.activeRoom$.subscribe(room => {
        if (room) {
          this.roomId = room.id;
          this.settingsName = room.name || '';
          this.settingsDescription = room.description || '';
          this.settingsAddMusicPolicy = room.add_music_policy || 'all';
          this.settingsAvatarUrl = room.avatar_url || '';
          this.settingsRules = room.rules || '';
          this.settingsTheme = room.theme || 'cool-ocean';
        }
      }),
      this.state.roomMemberRole$.subscribe(role => {
        this.isOwner = role === 'OWNER';
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  public onSaveSettingsSubmit(event: Event): void {
    event.preventDefault();
    if (!this.roomId) return;

    this.isSavingSettings = true;
    this.api.room.updateSettings(
      this.roomId,
      this.settingsName,
      this.settingsDescription,
      this.settingsAddMusicPolicy,
      this.settingsAvatarUrl || null,
      this.settingsRules || null,
      this.settingsTheme
    ).subscribe({
      next: (updatedRoom) => {
        this.isSavingSettings = false;
        this.toast.success('Cập nhật cấu hình phòng thành công.');
        this.state.activeRoom$.next(updatedRoom);
      },
      error: (err) => {
        this.isSavingSettings = false;
        this.toast.error('Cập nhật cấu hình thất bại: ' + (err.message || err));
      }
    });
  }

  public close(): void {
    this.uiState.toggleIdentity(false);
  }
}
