import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../shared/services/toast.service';

@Component({ selector: 'app-admin', standalone: true, imports: [CommonModule, FormsModule], templateUrl: './admin.component.html', styleUrl: './admin.component.css' })
export class AdminComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);
  public rooms: any[] = [];
  public accounts: any[] = [];
  public auditEvents: any[] = [];
  public members: any[] = [];
  public messages: any[] = [];
  public tracks: any[] = [];
  public membersLoading = false;
  public messagesLoading = false;
  public selected: any | null = null;
  public loading = true;
  public saving = false;

  async ngOnInit(): Promise<void> { await Promise.all([this.loadRooms(), this.loadAccounts(), this.loadAudit(), this.loadTracks()]); }
  public async loadTracks(): Promise<void> { try { this.tracks = await firstValueFrom(this.api.admin.listTracks()); } catch { this.tracks = []; } }
  public async removeTrack(track: any): Promise<void> { if (!window.confirm(`Xóa track ${track.title}?`)) return; try { await firstValueFrom(this.api.admin.deleteTrack(track.id)); this.tracks = this.tracks.filter(item => item.id !== track.id); } catch (error: any) { this.toast.error(error?.message || 'Không thể xóa track.'); } }
  public async loadRooms(): Promise<void> {
    this.loading = true;
    try { this.rooms = (await firstValueFrom(this.api.admin.listRooms())).map(room => this.normalizeRoom(room)); }
    catch (error: any) { this.toast.error(error?.message || 'Không thể tải dữ liệu quản trị.'); }
    finally { this.loading = false; }
  }
  public async select(room: any): Promise<void> {
    this.selected = { ...room };
    this.members = [];
    this.messages = [];
    this.membersLoading = true;
    this.messagesLoading = true;
    try { this.members = await firstValueFrom(this.api.admin.listRoomMembers(room.id)); }
    catch (error: any) { this.toast.error(error?.message || 'Không thể tải members.'); }
    finally { this.membersLoading = false; }
    try { this.messages = await firstValueFrom(this.api.admin.listRoomMessages(room.id)); }
    catch (error: any) { this.toast.error(error?.message || 'Không thể tải chat history.'); }
    finally { this.messagesLoading = false; }
  }
  public async loadAccounts(): Promise<void> {
    try { this.accounts = await firstValueFrom(this.api.admin.listAccounts()); }
    catch (error: any) { this.toast.error(error?.message || 'Không thể tải accounts.'); }
  }
  public async toggleAdmin(account: any): Promise<void> {
		try {
			const result = await firstValueFrom(this.api.admin.setAccountAdmin(account.id, !account.is_admin));
			account.is_admin = result.is_admin;
			this.toast.success('Đã cập nhật quyền account. Token mới áp dụng khi người dùng refresh hoặc đăng nhập lại.');
		} catch (error: any) { this.toast.error(error?.message || 'Không thể cập nhật quyền account.'); }
	}
  public async toggleSuspension(account: any): Promise<void> {
    const suspended = !account.is_suspended;
    const action = suspended ? 'Tạm ngưng' : 'Mở lại';
    if (!window.confirm(`${action} tài khoản ${account.username}?`)) return;
    try {
      const result = await firstValueFrom(this.api.admin.setAccountSuspended(account.id, suspended));
      account.is_suspended = result.is_suspended;
      this.toast.success(suspended ? 'Đã tạm ngưng tài khoản và thu hồi refresh sessions.' : 'Đã mở lại tài khoản.');
    } catch (error: any) { this.toast.error(error?.message || 'Không thể cập nhật trạng thái tài khoản.'); }
  }
  public async save(): Promise<void> {
    if (!this.selected || this.saving) return;
    this.saving = true;
    try {
      const updated = await firstValueFrom(this.api.admin.updateRoom(this.selected.id, this.selected));
      const normalized = this.normalizeRoom(updated);
      this.rooms = this.rooms.map(room => room.id === normalized.id ? normalized : room);
      this.selected = { ...normalized };
      this.toast.success('Đã lưu room.');
    } catch (error: any) { this.toast.error(error?.message || 'Không thể lưu room.'); }
    finally { this.saving = false; }
  }
  public async loadAudit(): Promise<void> { try { this.auditEvents = await firstValueFrom(this.api.admin.listAudit()); } catch { this.auditEvents = []; } }
  public async removeMember(member: any): Promise<void> {
    if (!this.selected || member.role_type === 'OWNER' || !window.confirm(`Gỡ ${member.user_id} khỏi room?`)) return;
    try {
      await firstValueFrom(this.api.admin.removeRoomMember(this.selected.id, member.user_id));
      this.members = this.members.filter(item => item.user_id !== member.user_id);
      await this.loadAudit();
      this.toast.success('Đã gỡ member khỏi room.');
    } catch (error: any) { this.toast.error(error?.message || 'Không thể gỡ member.'); }
  }
  public async removeMessage(message: any): Promise<void> {
    if (!this.selected || !window.confirm('Xóa tin nhắn này?')) return;
    try {
      await firstValueFrom(this.api.admin.deleteRoomMessage(this.selected.id, message.id));
      this.messages = this.messages.filter(item => item.id !== message.id);
      this.toast.success('Đã xóa tin nhắn.');
    } catch (error: any) { this.toast.error(error?.message || 'Không thể xóa tin nhắn.'); }
  }
  public async remove(): Promise<void> {
    if (!this.selected || !window.confirm(`Xóa room "${this.selected.name}"?`)) return;
    try {
      await firstValueFrom(this.api.admin.deleteRoom(this.selected.id));
      this.rooms = this.rooms.filter(room => room.id !== this.selected.id);
      this.selected = null;
      this.toast.success('Đã xóa room.');
    } catch (error: any) { this.toast.error(error?.message || 'Không thể xóa room.'); }
  }
  public back(): void { void this.router.navigate(['/dashboard']); }
  private normalizeRoom(room: any): any { return { ...room, privacy: String(room.privacy || 'public').toLowerCase() }; }
}
