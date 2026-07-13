import { CommonModule } from '@angular/common';
import { Component, Input, OnInit, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiService } from '../../../../core/services/api.service';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';

@Component({
  selector: 'app-room-session',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './session.component.html',
  styleUrl: './session.component.css'
})
export class SessionComponent implements OnInit {
  @Input() roomId = '';

  private api = inject(ApiService);
  private state = inject(StateService);
  private toast = inject(ToastService);

  public templates: any[] = [];
  public workspace: any | null = null;
  public selectedTemplate = 'focus';
  public title = 'Focus sprint';
  public goal = '';
  public agendaContent = '';
  public actionContent = '';
  public loading = true;
  public creating = false;

  public async ngOnInit(): Promise<void> {
    await Promise.all([this.loadTemplates(), this.loadActiveSession()]);
    this.loading = false;
  }

  public get canManage(): boolean {
    const role = this.state.roomMemberRole$.value;
    return role === 'OWNER' || role === 'MODERATOR';
  }

  public get session(): any | null { return this.workspace?.session || null; }

  public onTemplateChange(): void {
    const template = this.templates.find(item => item.key === this.selectedTemplate);
    if (template) this.title = template.name;
  }

  public async start(): Promise<void> {
    if (!this.canManage || !this.title.trim() || this.creating) return;
    this.creating = true;
    try {
      const session = await firstValueFrom(this.api.room.startSession(this.roomId, this.title.trim(), this.goal.trim(), this.selectedTemplate));
      await this.loadWorkspace(session.id);
      this.toast.success('Đã bắt đầu session cho cả phòng.');
    } catch (error: any) {
      this.toast.error(error?.message || 'Không thể bắt đầu session.');
    } finally {
      this.creating = false;
    }
  }

  public async complete(): Promise<void> {
    if (!this.canManage || !this.session) return;
    try {
      await firstValueFrom(this.api.room.completeSession(this.roomId, this.session.id));
      this.workspace = null;
      this.toast.success('Session đã kết thúc.');
    } catch (error: any) {
      this.toast.error(error?.message || 'Không thể kết thúc session.');
    }
  }

  public async addAgenda(): Promise<void> {
    if (!this.session || !this.agendaContent.trim()) return;
    try {
      await firstValueFrom(this.api.room.addAgendaItem(this.roomId, this.session.id, this.agendaContent.trim(), this.workspace.agenda.length));
      this.agendaContent = '';
      await this.loadWorkspace(this.session.id);
    } catch (error: any) { this.toast.error(error?.message || 'Không thể thêm agenda.'); }
  }

  public async toggleAgenda(item: any): Promise<void> {
    if (!this.session) return;
    try {
      await firstValueFrom(this.api.room.updateAgendaItem(this.roomId, this.session.id, item.id, !item.is_done));
      item.is_done = !item.is_done;
    } catch (error: any) { this.toast.error(error?.message || 'Không thể cập nhật agenda.'); }
  }

  public async addAction(): Promise<void> {
    if (!this.session || !this.actionContent.trim()) return;
    try {
      await firstValueFrom(this.api.room.addActionItem(this.roomId, this.session.id, this.actionContent.trim()));
      this.actionContent = '';
      await this.loadWorkspace(this.session.id);
    } catch (error: any) { this.toast.error(error?.message || 'Không thể thêm action item.'); }
  }

  public async toggleAction(item: any): Promise<void> {
    if (!this.session) return;
    const status = item.status === 'DONE' ? 'OPEN' : 'DONE';
    try {
      await firstValueFrom(this.api.room.updateActionItem(this.roomId, this.session.id, item.id, status));
      item.status = status;
    } catch (error: any) { this.toast.error(error?.message || 'Không thể cập nhật action item.'); }
  }

  private async loadTemplates(): Promise<void> {
    try { this.templates = await firstValueFrom(this.api.room.listSessionTemplates()); } catch { this.templates = []; }
  }

  private async loadActiveSession(): Promise<void> {
    try {
      const session = await firstValueFrom(this.api.room.getActiveSession(this.roomId));
      if (session) await this.loadWorkspace(session.id);
    } catch (error: any) {
      if (error?.message) this.toast.error('Không thể tải session hiện tại.');
    }
  }

  private async loadWorkspace(sessionId: string): Promise<void> {
    this.workspace = await firstValueFrom(this.api.room.getSessionWorkspace(this.roomId, sessionId));
  }
}
