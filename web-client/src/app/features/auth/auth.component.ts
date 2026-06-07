import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { StateService } from '../../core/services/state.service';
import { ToastService } from '../../shared/services/toast.service';

@Component({
  selector: 'app-auth',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './auth.component.html',
  styleUrl: './auth.component.css'
})
export class AuthComponent implements OnInit {
  private api = inject(ApiService);
  private state = inject(StateService);
  private toast = inject(ToastService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  public isRegister = false;
  public username = '';
  public identity = '';
  public password = '';
  public isLoading = false;
  public pwStrengthClass = '';

  ngOnInit(): void {
    // Xử lý redirect sau verify email (/auth?verified=true)
    // và lỗi Google OAuth (/auth?error=xxx)
    this.route.queryParams.subscribe(params => {
      if (params['verified'] === 'true') {
        this.toast.success('✅ Email đã được xác thực! Bạn có thể đăng nhập.');
      }
      if (params['error']) {
        const msgs: Record<string, string> = {
          'invalid_state': 'Lỗi bảo mật, vui lòng thử lại.',
          'no_code': 'Không nhận được mã từ Google.',
          'google_exchange_failed': 'Không thể xác thực với Google.',
          'login_failed': 'Đăng nhập thất bại, vui lòng thử lại.',
        };
        this.toast.error(msgs[params['error']] || 'Đã có lỗi xảy ra.');
      }
    });
  }

  public toggleMode(): void {
    this.isRegister = !this.isRegister;
    this.username = '';
    this.identity = '';
    this.password = '';
    this.pwStrengthClass = '';
  }

  public onPasswordInput(val: string): void {
    if (!this.isRegister) return;
    if (val.length === 0) {
      this.pwStrengthClass = '';
    } else if (val.length < 6) {
      this.pwStrengthClass = 'strength-weak';
    } else if (val.length < 10) {
      this.pwStrengthClass = 'strength-medium';
    } else {
      const hasUpper = /[A-Z]/.test(val);
      const hasNumber = /[0-9]/.test(val);
      this.pwStrengthClass = (hasUpper && hasNumber) ? 'strength-strong' : 'strength-medium';
    }
  }

  public async onSubmit(event: Event): Promise<void> {
    event.preventDefault();
    if (this.isLoading) return;
    this.isLoading = true;

    try {
      if (this.isRegister) {
        await this.api.auth.register(this.username, this.identity, this.password).toPromise();
        this.toast.success('Đăng ký thành công! Kiểm tra email để xác thực tài khoản.');
        this.isRegister = false;
        this.username = '';
        this.identity = '';
        this.password = '';
        this.pwStrengthClass = '';
      } else {
        const loginData = await this.api.auth.login(this.identity, this.password).toPromise();
        this.state.accessToken$.next(loginData.access_token);

        const profile = await this.api.user.getProfile(loginData.user_id).toPromise();
        this.state.user$.next(profile);

        this.toast.success(`Chào mừng quay lại, ${profile?.display_name || loginData.username}!`);
        this.router.navigate(['/dashboard']);
      }
    } catch (err: any) {
      this.toast.error(err.message || 'Xác thực thất bại.');
    } finally {
      this.isLoading = false;
    }
  }

  public loginWithGoogle(): void {
    // Redirect thật sang Google OAuth endpoint
    window.location.href = 'http://localhost:8080/api/v1/auth/google';
  }
}
