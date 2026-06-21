import { Injectable, inject } from '@angular/core';
import { StateService } from './state.service';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class NotificationStreamService {
  private state = inject(StateService);
  private stream: EventSource | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private isStopped = false;
  private onNotification: ((notification: any) => void) | null = null;

  public connect(token: string, onNotification: (notification: any) => void): void {
    if (!token) {
      return;
    }

    this.disconnect();
    this.isStopped = false;
    this.onNotification = onNotification;
    this.openStream(token);
  }

  public disconnect(): void {
    this.isStopped = true;
    this.onNotification = null;

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.stream) {
      this.stream.close();
      this.stream = null;
    }
  }

  private openStream(token?: string): void {
    const effectiveToken = token || this.state.accessToken;
    if (!effectiveToken || !this.onNotification) {
      return;
    }

    const url = this.buildStreamUrl(effectiveToken);
    this.stream = new EventSource(url);

    this.stream.addEventListener('notification', (event: MessageEvent) => {
      try {
        const payload = JSON.parse(event.data);
        this.onNotification?.(payload);
      } catch (error) {
        console.warn('Failed to parse notification payload:', error);
      }
    });

    this.stream.onerror = () => {
      if (this.stream) {
        this.stream.close();
        this.stream = null;
      }

      if (!this.isStopped) {
        this.scheduleReconnect();
      }
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    this.reconnectTimer = setTimeout(() => {
      if (!this.isStopped) {
        this.openStream();
      }
    }, 3000);
  }

  private buildStreamUrl(token: string): string {
    let baseUrl = environment.apiUrl;
    if (!baseUrl.startsWith('http://') && !baseUrl.startsWith('https://')) {
      const protocol = window.location.protocol;
      const host = window.location.host;
      const separator = baseUrl.startsWith('/') ? '' : '/';
      baseUrl = `${protocol}//${host}${separator}${baseUrl}`;
    }
    return `${baseUrl}/notifications/stream?token=${encodeURIComponent(token)}`;
  }
}
