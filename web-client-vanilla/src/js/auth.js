import { API } from './api.js';
import { State } from './state.js';
import { Toast } from './components/toast.js';

export const AuthView = {
  container: null,

  init(container) {
    this.container = container;
  },

  render(isRegister = false) {
    this.container.innerHTML = `
      <div class="auth-container">
        <!-- Brand Panel (Left) -->
        <div class="auth-brand-panel">
          <div class="brand-header">
            <span style="font-weight: 800; font-size: 20px; background-color: var(--accent-primary); color: black; padding: 2px 8px; border-radius: var(--radius-sm);">WT</span>
            <span class="brand-name">WorkTogether</span>
          </div>
          <div class="brand-essence-wrap">
            <h1>Cùng chia sẻ không gian âm nhạc & làm việc nhóm.</h1>
            <p>Nền tảng đồng bộ âm thanh thời gian thực, voice call tích hợp và chat nhóm bảo mật với hiệu năng cao cho đội ngũ phát triển.</p>
          </div>
          <div class="brand-footer">
            © 2026 WorkTogether. Thiết kế tối giản tối ưu năng suất.
          </div>
        </div>

        <!-- Form Panel (Right) -->
        <div class="auth-form-panel">
          <div class="auth-card">
            <div class="auth-card-header">
              <h2>${isRegister ? 'Tạo tài khoản mới' : 'Chào mừng quay trở lại'}</h2>
              <p>${isRegister ? 'Điền thông tin bên dưới để đăng ký.' : 'Đăng nhập vào không gian làm việc của bạn.'}</p>
            </div>

            <form id="auth-form" class="auth-form">
              ${isRegister ? `
                <div class="form-group">
                  <label class="form-label" for="reg-username">Tên người dùng</label>
                  <input type="text" id="reg-username" placeholder="vd: coder_pro" required>
                </div>
              ` : ''}
              
              <div class="form-group">
                <label class="form-label" for="auth-identity">${isRegister ? 'Địa chỉ Email' : 'Tên đăng nhập hoặc Email'}</label>
                <input type="${isRegister ? 'email' : 'text'}" id="auth-identity" placeholder="${isRegister ? 'email@example.com' : 'username_hoac_email'}" required>
              </div>

              <div class="form-group">
                <label class="form-label" for="auth-password">Mật khẩu</label>
                <input type="password" id="auth-password" placeholder="••••••••" required>
                ${isRegister ? `
                  <div class="pw-strength-bar">
                    <div id="pw-strength-fill" class="pw-strength-fill"></div>
                  </div>
                ` : ''}
              </div>

              <button type="submit" class="btn btn-primary w-full" style="padding: 12px; margin-top: 8px;">
                ${isRegister ? 'Đăng ký Tài khoản' : 'Đăng nhập Workspace'}
              </button>
            </form>

            <div class="divider">Hoặc tiếp tục với</div>
            <button id="google-login-btn" class="btn btn-google w-full">
              <svg class="google-icon" viewBox="0 0 24 24" width="16" height="16">
                <path fill="#EA4335" d="M12.24 10.285V14.4h6.887c-.648 2.41-2.519 4.114-5.136 4.114A5.79 5.79 0 0 1 8.2 12.725a5.79 5.79 0 0 1 5.79-5.79c1.496 0 2.85.55 3.886 1.454l3.242-3.242C19.143 3.332 16.793 2.4 13.99 2.4 8.14 2.4 3.4 7.14 3.4 12.99s4.74 10.59 10.59 10.59c6.07 0 10.09-4.27 10.09-10.27 0-.69-.06-1.35-.18-2.025H12.24Z"/>
              </svg>
              Google Account
            </button>

            <p class="form-toggle-link">
              ${isRegister ? 'Đã có tài khoản?' : 'Chưa có tài khoản?'} 
              <a href="#" id="toggle-auth-link">${isRegister ? 'Đăng nhập ngay' : 'Đăng ký miễn phí'}</a>
            </p>
          </div>
        </div>
      </div>
    `;

    this.bindEvents(isRegister);
  },

  bindEvents(isRegister) {
    const form = document.getElementById('auth-form');
    const toggleLink = document.getElementById('toggle-auth-link');
    const googleBtn = document.getElementById('google-login-btn');

    // Toggle between login and register
    toggleLink.onclick = (e) => {
      e.preventDefault();
      window.location.hash = isRegister ? '#/auth/login' : '#/auth/register';
    };

    // Google Login Placeholder
    googleBtn.onclick = () => {
      Toast.info("Đang kết nối tới máy chủ xác thực Google OAuth...");
      // Trong môi trường thật, chuyển hướng đến API Gateway endpoint:
      // window.location.href = "http://localhost:8080/api/v1/auth/google";
    };

    // Password strength visualizer
    if (isRegister) {
      const pwInput = document.getElementById('auth-password');
      const strengthFill = document.getElementById('pw-strength-fill');
      
      pwInput.oninput = () => {
        const val = pwInput.value;
        strengthFill.className = 'pw-strength-fill';
        
        if (val.length === 0) {
          strengthFill.style.width = '0%';
        } else if (val.length < 6) {
          strengthFill.classList.add('strength-weak');
        } else if (val.length < 10) {
          strengthFill.classList.add('strength-medium');
        } else {
          // Check uppercase and numbers for strong
          const hasUpper = /[A-Z]/.test(val);
          const hasNumber = /[0-9]/.test(val);
          if (hasUpper && hasNumber) {
            strengthFill.classList.add('strength-strong');
          } else {
            strengthFill.classList.add('strength-medium');
          }
        }
      };
    }

    // Form submit
    form.onsubmit = async (e) => {
      e.preventDefault();

      const submitBtn = form.querySelector('button[type="submit"]');
      submitBtn.disabled = true;
      const originalText = submitBtn.innerText;
      submitBtn.innerText = isRegister ? 'Đang tạo tài khoản...' : 'Đang xác thực...';

      try {
        if (isRegister) {
          const username = document.getElementById('reg-username').value;
          const email = document.getElementById('auth-identity').value;
          const password = document.getElementById('auth-password').value;
          
          await API.auth.register(username, email, password);
          Toast.success("Đăng ký thành công! Hãy kiểm tra hòm thư kích hoạt.");
          window.location.hash = '#/auth/login';
        } else {
          const identity = document.getElementById('auth-identity').value;
          const password = document.getElementById('auth-password').value;
          
          const loginData = await API.auth.login(identity, password);
          State.set('accessToken', loginData.access_token);
          
          // Lấy thông tin profile ngay sau khi login thành công
          const profile = await API.user.getProfile(loginData.user_id || "me");
          State.set('user', profile);
          
          Toast.success(`Chào mừng quay lại, ${profile.display_name}!`);
          window.location.hash = '#/dashboard';
        }
      } catch (err) {
        Toast.error(err.message || "Xác thực thất bại.");
        submitBtn.disabled = false;
        submitBtn.innerText = originalText;
      }
    };
  }
};
