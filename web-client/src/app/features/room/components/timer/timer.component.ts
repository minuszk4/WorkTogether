import { Component, Input, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { StateService } from '../../../../core/services/state.service';
import { ToastService } from '../../../../shared/services/toast.service';
import { Subscription } from 'rxjs';
import { environment } from '../../../../../environments/environment';

export interface PomodoroState {
  status: 'focus' | 'break' | 'paused' | 'idle';
  paused_status?: 'focus' | 'break';
  duration_seconds: number;
  break_seconds: number;
  remaining_seconds: number;
  ends_at: number;
  current_cycle: number;
  total_cycles: number;
}

@Component({
  selector: 'app-room-timer',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './timer.component.html',
  styleUrl: './timer.component.css'
})
export class TimerComponent implements OnInit, OnDestroy {
  public state = inject(StateService);
  private toast = inject(ToastService);

  @Input() roomId = '';

  private ws: WebSocket | null = null;
  public timerState: PomodoroState = {
    status: 'idle',
    duration_seconds: 1500,
    break_seconds: 300,
    remaining_seconds: 0,
    ends_at: 0,
    current_cycle: 0,
    total_cycles: 0
  };

  // User input settings
  public focusMinutes = 25;
  public breakMinutes = 5;
  public cyclesCount = 4;

  private subs: Subscription[] = [];
  private reconnectTimeout: any = null;

  ngOnInit(): void {
    this.connectWs();
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    this.disconnectWs();
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
    }
  }

  private connectWs(): void {
    if (!this.roomId) return;

    const token = this.state.accessToken;
    if (!token) return;

    const wsUrl = `${environment.wsUrl}/api/v1/rooms/${this.roomId}/timer/ws?token=${token}`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onmessage = (event) => {
      try {
        const rawMsg = JSON.parse(event.data);
        if (rawMsg.event === 'timer:sync' && rawMsg.payload) {
          this.timerState = rawMsg.payload;
        }
      } catch (err) {
        console.warn('Lỗi xử lý WS timer:', err);
      }
    };

    this.ws.onclose = () => {
      console.log('WS timer closed, reconnecting...');
      this.reconnectTimeout = setTimeout(() => this.connectWs(), 3000);
    };

    this.ws.onerror = (err) => {
      console.error('WS timer error:', err);
    };
  }

  private disconnectWs(): void {
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
  }

  private sendCommand(event: string, payload?: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.toast.error('Mất kết nối máy chủ timer.');
      return;
    }
    const msg = {
      event: event,
      room_id: this.roomId,
      payload: payload ? JSON.stringify(payload) : null
    };
    this.ws.send(JSON.stringify(msg));
  }

  public onStartTimer(): void {
    if (this.focusMinutes <= 0 || this.breakMinutes <= 0 || this.cyclesCount <= 0) {
      this.toast.error('Vui lòng cấu hình thời gian hợp lệ.');
      return;
    }

    const durationSeconds = this.focusMinutes * 60;
    const breakSeconds = this.breakMinutes * 60;

    this.sendCommand('timer:start', {
      duration_seconds: durationSeconds,
      break_seconds: breakSeconds,
      cycles: this.cyclesCount
    });
    this.toast.success('Đã bắt đầu Pomodoro toàn phòng!');
  }

  public onPauseTimer(): void {
    this.sendCommand('timer:pause');
    this.toast.info('Đã tạm dừng đếm ngược.');
  }

  public onResumeTimer(): void {
    this.sendCommand('timer:resume');
    this.toast.success('Tiếp tục đếm ngược.');
  }

  public onStopTimer(): void {
    this.sendCommand('timer:stop');
    this.toast.info('Đã dừng Pomodoro.');
  }

  // Display helpers
  public get formattedTime(): string {
    const totalSecs = this.timerState.status === 'idle' 
      ? this.focusMinutes * 60 
      : this.timerState.remaining_seconds;
    const mins = Math.floor(totalSecs / 60);
    const secs = totalSecs % 60;
    return `${this.padZero(mins)}:${this.padZero(secs)}`;
  }

  public get progressPct(): number {
    if (this.timerState.status === 'idle') return 100;
    const total = this.timerState.status === 'break' 
      ? this.timerState.break_seconds 
      : this.timerState.duration_seconds;
    if (total <= 0) return 0;
    return (this.timerState.remaining_seconds / total) * 100;
  }

  public get progressStrokeDashoffset(): number {
    // 2 * PI * r = 2 * 3.14159 * 40 = 251.2
    const circumference = 251.2;
    const pct = this.progressPct;
    return circumference - (pct / 100) * circumference;
  }

  private padZero(n: number): string {
    return n < 10 ? '0' + n : '' + n;
  }

  public get isModeratorOrOwner(): boolean {
    const role = this.state.roomMemberRole;
    return role === 'OWNER' || role === 'MODERATOR';
  }
}
