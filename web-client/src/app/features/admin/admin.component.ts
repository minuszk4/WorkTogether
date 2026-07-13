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
  public selected: any | null = null;
  public loading = true;
  public saving = false;

  async ngOnInit(): Promise<void> { await this.loadRooms(); }
  public async loadRooms(): Promise<void> {
    this.loading = true;
    try { this.rooms = (await firstValueFrom(this.api.admin.listRooms())).map(room => this.normalizeRoom(room)); }
    catch (error: any) { this.toast.error(error?.message || 'Không thể tải dữ liệu quản trị.'); }
    finally { this.loading = false; }
  }
  public select(room: any): void { this.selected = { ...room }; }
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
