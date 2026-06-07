import { State } from './state.js';
import { API } from './api.js';
import { AuthView } from './auth.js';
import { DashboardView } from './dashboard.js';
import { RoomView } from './room.js';
import { Toast } from './components/toast.js';

class App {
  constructor() {
    this.viewContainer = null;
  }

  async start() {
    this.viewContainer = document.getElementById('app-root');
    Toast.init();

    // Lắng nghe sự thay đổi hash của URL để điều hướng
    window.addEventListener('hashchange', () => this.handleRoute());

    // 1. Silent Login check (Làm mới token khi mở trang)
    const authed = await this.trySilentLogin();
    
    // 2. Chạy Router ban đầu
    this.handleRoute(authed);
  }

  async trySilentLogin() {
    try {
      // Thử gọi refresh API để lấy token mới
      // Nginx routes post /auth/refresh -> http-only cookie sẽ tự động gửi kèm
      const response = await fetch('http://localhost:8080/api/v1/auth/refresh', {
        method: 'POST',
        credentials: 'include'
      });
      
      const data = await response.json();
      if (response.ok && data.success && data.data.access_token) {
        State.set('accessToken', data.data.access_token);
        
        // Lấy thông tin user profile
        const profile = await API.user.getProfile("me");
        State.set('user', profile);
        
        console.log("[App Init] Silent login thành công cho user:", profile.display_name);
        return true;
      }
    } catch (err) {
      console.log("[App Init] Không có phiên làm việc cũ hoạt động.");
    }
    return false;
  }

  async handleRoute(forceAuthed = false) {
    const hash = window.location.hash || '#/auth/login';
    
    // Phân tích Route
    const isRegister = hash === '#/auth/register';
    const isLogin = hash === '#/auth/login' || hash === '#/auth';
    const isDashboard = hash === '#/dashboard';
    const isRoom = hash.startsWith('#/room/');

    const user = State.get('user');
    const token = State.get('accessToken');
    const hasSession = user && token;

    // Route Guards (Bảo vệ tuyến đường)
    if ((isDashboard || isRoom) && !hasSession && !forceAuthed) {
      // Nếu cố vào trang chính mà chưa login -> Về login
      window.location.hash = '#/auth/login';
      return;
    }

    if ((isLogin || isRegister) && hasSession) {
      // Nếu đã login mà cố vào trang auth -> Vào thẳng dashboard
      window.location.hash = '#/dashboard';
      return;
    }

    // Render Views tương ứng
    try {
      if (isLogin) {
        AuthView.init(this.viewContainer);
        AuthView.render(false);
      } else if (isRegister) {
        AuthView.init(this.viewContainer);
        AuthView.render(true);
      } else if (isDashboard) {
        // Dọn dẹp connection phòng cũ nếu có
        RoomView.disconnectAll();
        
        DashboardView.init(this.viewContainer);
        await DashboardView.render();
      } else if (isRoom) {
        const roomId = hash.split('/')[2];
        RoomView.init(this.viewContainer);
        await RoomView.render(roomId);
      }
    } catch (err) {
      console.log("Lỗi Router:", err);
      Toast.error("Không thể tải trang: " + err.message);
      window.location.hash = '#/dashboard';
    }
  }
}

// Chờ DOM sẵn sàng để khởi động
document.addEventListener('DOMContentLoaded', () => {
  const app = new App();
  app.start();
});
