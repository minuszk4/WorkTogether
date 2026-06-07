import { Component, OnInit, inject } from '@angular/core';
import { Router } from '@angular/router';
import { StateService } from '../../core/services/state.service';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../shared/services/toast.service';
import { CommonModule } from '@angular/common';

/**
 * AuthCallbackComponent xử lý redirect sau Google OAuth.
 * Backend redirect về: /auth/callback#access_token=xxx&user_id=yyy&username=zzz
 * Component này parse fragment, lưu token và điều hướng về dashboard.
 */
@Component({
  selector: 'app-auth-callback',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div style="display:flex;align-items:center;justify-content:center;height:100vh;background:#0d0d0d;color:#fff;flex-direction:column;gap:16px;">
      <div style="width:40px;height:40px;border:3px solid #7c3aed;border-top-color:transparent;border-radius:50%;animation:spin 0.8s linear infinite;"></div>
      <p style="color:#9ca3af;font-size:14px;">Đang xác thực với Google...</p>
      <style>@keyframes spin{to{transform:rotate(360deg)}}</style>
    </div>
  `
})
export class AuthCallbackComponent implements OnInit {
  private state = inject(StateService);
  private api = inject(ApiService);
  private router = inject(Router);
  private toast = inject(ToastService);

  ngOnInit(): void {
    this.handleCallback();
  }

  private async handleCallback(): Promise<void> {
    try {
      // Parse URL fragment: #access_token=xxx&user_id=yyy&username=zzz
      const fragment = window.location.hash.substring(1);
      const params = new URLSearchParams(fragment);

      const accessToken = params.get('access_token');
      const userId = params.get('user_id');
      const username = params.get('username');

      if (!accessToken || !userId) {
        this.toast.error('Đăng nhập Google thất bại — không nhận được token.');
        this.router.navigate(['/auth']);
        return;
      }

      // Lưu access token
      this.state.accessToken$.next(accessToken);

      // Fetch profile
      try {
        const profile = await this.api.user.getProfile(userId).toPromise();
        this.state.user$.next(profile);
        this.toast.success(`Chào mừng, ${profile?.display_name || username}! 🎵`);
      } catch {
        // Profile chưa tạo (tài khoản mới qua Google) — tạo với display_name từ username
        this.state.user$.next({ id: userId, username, display_name: username });
        this.toast.success(`Chào mừng đến WorkTogether, ${username}! 🎵`);
      }

      // Xóa fragment khỏi URL trước khi navigate
      window.history.replaceState({}, document.title, window.location.pathname);
      this.router.navigate(['/dashboard']);
    } catch (err: any) {
      this.toast.error('Lỗi xử lý xác thực Google.');
      this.router.navigate(['/auth']);
    }
  }
}
