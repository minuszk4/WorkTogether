import { Component, Input, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { SubroomService } from '../../../../core/services/subroom.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { Subscription } from 'rxjs';

export interface SubRoom {
  id: string;
  name: string;
  description: string;
  parent_id: string;
}

@Component({
  selector: 'app-room-subrooms',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './subrooms.component.html',
  styleUrl: './subrooms.component.css'
})
export class SubroomsComponent implements OnInit, OnDestroy {
  private subroomService = inject(SubroomService);
  public state = inject(StateService);
  private toast = inject(ToastService);

  @Input() roomId = '';

  public subrooms: SubRoom[] = [];
  public showCreateForm = false;
  public newSubroomName = '';
  public newSubroomDesc = '';
  public isCreating = false;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.loadSubrooms();
    // Refresh subrooms list every 10 seconds to keep synced
    const intervalId = setInterval(() => this.loadSubrooms(), 10000);
    this.subs.push(new Subscription(() => clearInterval(intervalId)));
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  public loadSubrooms(): void {
    if (!this.roomId) return;
    this.subroomService.getSubrooms(this.roomId).subscribe({
      next: (data) => {
        this.subrooms = data || [];
      },
      error: (err) => {
        console.warn('Lỗi lấy danh sách phòng con:', err);
      }
    });
  }

  public toggleCreateForm(): void {
    this.showCreateForm = !this.showCreateForm;
    this.newSubroomName = '';
    this.newSubroomDesc = '';
  }

  public onCreateSubmit(): void {
    if (!this.newSubroomName.trim()) {
      this.toast.error('Tên phòng con không được để trống.');
      return;
    }
    this.isCreating = true;
    this.subroomService.createSubroom(this.roomId, this.newSubroomName.trim(), this.newSubroomDesc.trim()).subscribe({
      next: (newRoom) => {
        this.toast.success(`Đã tạo phòng con "${newRoom.name}" thành công.`);
        this.subrooms.push(newRoom);
        this.toggleCreateForm();
        this.isCreating = false;
      },
      error: (err) => {
        this.toast.error('Lỗi tạo phòng con: ' + err.message);
        this.isCreating = false;
      }
    });
  }

  public getMembersInSubroom(subroomId: string | null): any[] {
    const list = this.state.activeRoomMembers$.value || [];
    return list.filter(m => {
      if (subroomId === null) {
        return !m.active_sub_room_id;
      }
      return m.active_sub_room_id === subroomId;
    });
  }

  public get currentMember(): any {
    const list = this.state.activeRoomMembers$.value || [];
    return list.find(m => m.isCurrentUser);
  }

  public get isUserInSubroom(): boolean {
    return !!this.currentMember?.active_sub_room_id;
  }

  public isUserInThisSubroom(subroomId: string): boolean {
    return this.currentMember?.active_sub_room_id === subroomId;
  }

  public onJoinLeaveSubroom(subroom: SubRoom): void {
    const currentUserId = this.state.user?.id;
    if (!currentUserId) return;

    const targetSubRoomId = this.isUserInThisSubroom(subroom.id) ? null : subroom.id;
    this.toast.info(targetSubRoomId ? 'Đang vào phòng con...' : 'Đang quay lại sảnh chính...');

    this.subroomService.moveMember(this.roomId, currentUserId, targetSubRoomId).subscribe({
      next: () => {
        this.toast.success(targetSubRoomId ? `Đã tham gia phòng con "${subroom.name}"` : 'Đã quay lại sảnh chính.');
        // Trigger manual refresh of members in StateService
        // Note: the room component will capture the change in members list and auto-reconnect voice!
        this.loadSubrooms();
      },
      error: (err) => {
        this.toast.error('Lỗi di chuyển: ' + err.message);
      }
    });
  }

  public onMoveMember(member: any, subroomId: string | null): void {
    this.subroomService.moveMember(this.roomId, member.user_id, subroomId).subscribe({
      next: () => {
        const dest = subroomId ? 'phòng con' : 'sảnh chính';
        this.toast.success(`Đã di chuyển ${member.display_name} vào ${dest}.`);
        this.loadSubrooms();
      },
      error: (err) => {
        this.toast.error('Lỗi di chuyển thành viên: ' + err.message);
      }
    });
  }

  public get isModeratorOrOwner(): boolean {
    const role = this.state.roomMemberRole;
    return role === 'OWNER' || role === 'MODERATOR';
  }
}
