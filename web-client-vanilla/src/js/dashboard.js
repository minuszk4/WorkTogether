import { API } from './api.js';
import { State } from './state.js';
import { Toast } from './components/toast.js';
import { Modal } from './components/modal.js';

export const DashboardView = {
  container: null,
  statusTimer: null,

  init(container) {
    this.container = container;
  },

  async render() {
    const user = State.get('user');
    if (!user) {
      window.location.hash = '#/auth';
      return;
    }

    this.container.innerHTML = `
      <div class="dashboard-layout">
        <!-- Slim Left Navigation (64px) -->
        <div class="dashboard-sidebar-slim">
          <div class="flex flex-col gap-4 items-center">
            <div class="slim-nav-item active" title="Lobby">
              <span style="font-weight: 800; font-size: 15px;">L</span>
            </div>
          </div>
          <div class="flex flex-col gap-4 items-center">
            <div class="slim-avatar" id="slim-avatar-btn" title="Hồ sơ cá nhân">
              ${user.avatar_url ? `<img src="${user.avatar_url}" alt="Avatar">` : `<span style="font-weight:600; font-size:12px;">${user.display_name.slice(0, 2).toUpperCase()}</span>`}
            </div>
            <div class="slim-nav-item" id="slim-logout-btn" title="Đăng xuất">
              <span style="font-size: 14px;">➔</span>
            </div>
          </div>
        </div>

        <!-- Main Workspace -->
        <div class="dashboard-main">
          <div class="dashboard-header">
            <div>
              <h1>WorkTogether Lobby</h1>
              <p style="font-size:13.5px; color:var(--text-secondary); margin-top:4px;">Tìm không gian làm việc hoặc tạo phòng cộng tác riêng của bạn.</p>
            </div>
            
            <!-- Presence Status Selector -->
            <div class="user-status-widget" id="status-widget">
              <div class="presence-dot online" id="status-dot"></div>
              <span class="presence-text" id="status-text">online</span>
            </div>
          </div>

          <!-- Rooms Content -->
          <div class="rooms-section">
            <div class="section-title-bar">
              <h2>Các phòng đang hoạt động</h2>
              <button class="btn btn-primary" id="open-create-room-btn">+ Tạo phòng mới</button>
            </div>

            <!-- Rooms list grid -->
            <div class="rooms-grid" id="rooms-grid-container">
              <!-- Loading Skeletons -->
              <div class="room-card skeleton"></div>
              <div class="room-card skeleton"></div>
              <div class="room-card skeleton"></div>
            </div>
          </div>
        </div>

        <!-- Right Friends Sidebar -->
        <div class="dashboard-friends-panel">
          <!-- Pending Requests Banner -->
          <div class="friend-requests-bar" id="pending-requests-bar" style="display: none;">
            <span>Yêu cầu kết bạn mới</span>
            <span class="friend-requests-badge" id="pending-requests-count">0</span>
          </div>

          <div class="friends-header">
            <h3>Đang trực tuyến</h3>
          </div>
          
          <div class="friends-list" id="friends-list-container">
            <p style="font-size:12.5px; color:var(--text-muted); text-align:center; margin-top:20px;">Đang tải danh sách...</p>
          </div>

          <!-- Add Friend form -->
          <div class="add-friend-section">
            <span style="font-size: 12px; font-weight: 600; color: var(--text-secondary);">Thêm bạn bè</span>
            <div style="display: flex; gap: 6px;">
              <input type="text" id="add-friend-id" placeholder="ID người dùng..." style="padding: 6px 10px; font-size:12.5px;">
              <button class="btn btn-primary" id="add-friend-btn" style="padding: 6px 10px;">Gửi</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Create Room Modal -->
      <div class="modal-overlay" id="create-room-modal">
        <div class="modal-container">
          <div class="modal-header">
            <h3>Tạo Phòng Cộng tác</h3>
            <button class="modal-close">&times;</button>
          </div>
          <div class="modal-body">
            <form id="create-room-form" class="flex flex-col gap-3">
              <div class="form-group">
                <label class="form-label" for="room-name">Tên phòng</label>
                <input type="text" id="room-name" placeholder="vd: Lofi Chill Study" required>
              </div>
              <div class="form-group">
                <label class="form-label" for="room-desc">Mô tả ngắn</label>
                <textarea id="room-desc" placeholder="Cùng nghe nhạc và làm việc nhóm..." rows="2"></textarea>
              </div>
              <div class="form-group">
                <label class="form-label" for="room-privacy">Quyền riêng tư</label>
                <select id="room-privacy" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 8px 12px; border-radius: var(--radius-md); outline: none;">
                  <option value="public">Công khai (Public)</option>
                  <option value="private">Riêng tư (Private - Có mật khẩu)</option>
                </select>
              </div>
              <div class="form-group" id="room-password-group" style="display: none;">
                <label class="form-label" for="room-password">Mật khẩu phòng</label>
                <input type="password" id="room-password" placeholder="Mật khẩu vào phòng">
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" onclick="document.querySelector('.modal-close').click()">Hủy bỏ</button>
            <button class="btn btn-primary" id="submit-create-room-btn">Khởi tạo</button>
          </div>
        </div>
      </div>

      <!-- Settings Profile Modal -->
      <div class="modal-overlay" id="profile-settings-modal">
        <div class="modal-container">
          <div class="modal-header">
            <h3>Cấu hình Hồ sơ cá nhân</h3>
            <button class="modal-close">&times;</button>
          </div>
          <div class="modal-body flex flex-col gap-4">
            <div style="display: flex; flex-direction: column; align-items: center; gap: 8px;">
              <div class="slim-avatar" style="width: 64px; height: 64px; border-width: 2px; font-size: 24px;" id="profile-avatar-preview">
                ${user.avatar_url ? `<img src="${user.avatar_url}" alt="Avatar">` : user.display_name.slice(0, 2).toUpperCase()}
              </div>
              <input type="file" id="avatar-file-input" accept="image/*" style="display: none;">
              <button class="btn btn-secondary" style="padding: 4px 10px; font-size:12px;" onclick="document.getElementById('avatar-file-input').click()">Thay đổi ảnh</button>
            </div>
            <div class="form-group">
              <label class="form-label" for="profile-displayname">Tên hiển thị</label>
              <input type="text" id="profile-displayname" value="${user.display_name}" required>
            </div>
            <div class="form-group">
              <label class="form-label" for="profile-bio">Mô tả ngắn</label>
              <textarea id="profile-bio" rows="3">${user.bio || ''}</textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" onclick="document.querySelector('#profile-settings-modal .modal-close').click()">Hủy</button>
            <button class="btn btn-primary" id="save-profile-btn">Lưu cấu hình</button>
          </div>
        </div>
      </div>
    `;

    this.bindEvents();
    await this.loadRooms();
    await this.loadFriends();
    
    // Bật heartbeat báo presence hoạt động
    this.startPresenceHeartbeat();
  },

  bindEvents() {
    const user = State.get('user');
    
    // Logout
    document.getElementById('slim-logout-btn').onclick = async () => {
      try {
        await API.auth.logout();
        State.clear();
        this.stopPresenceHeartbeat();
        window.location.hash = '#/auth';
        Toast.success("Đăng xuất thành công.");
      } catch (err) {
        Toast.error("Lỗi đăng xuất.");
      }
    };

    // Open profile modal
    document.getElementById('slim-avatar-btn').onclick = () => {
      Modal.open('profile-settings-modal');
    };

    // Open create room modal
    document.getElementById('open-create-room-btn').onclick = () => {
      Modal.open('create-room-modal');
    };

    // Toggle password field in create room form based on privacy choice
    const privacySelect = document.getElementById('room-privacy');
    const pwGroup = document.getElementById('room-password-group');
    privacySelect.onchange = () => {
      pwGroup.style.display = privacySelect.value === 'private' ? 'flex' : 'none';
    };

    // Create room submit
    document.getElementById('submit-create-room-btn').onclick = async () => {
      const name = document.getElementById('room-name').value;
      const desc = document.getElementById('room-desc').value;
      const privacy = privacySelect.value;
      const password = document.getElementById('room-password').value;

      if (!name) {
        Toast.error("Tên phòng không được bỏ trống.");
        return;
      }

      try {
        const room = await API.room.create(name, desc, privacy, password);
        Modal.close('create-room-modal');
        Toast.success(`Khởi tạo phòng "${room.name}" thành công.`);
        
        // Auto-join the newly created room
        await this.joinRoom(room.id, password);
      } catch (err) {
        Toast.error(err.message || "Không thể tạo phòng.");
      }
    };

    // Add friend submit
    document.getElementById('add-friend-btn').onclick = async () => {
      const friendId = document.getElementById('add-friend-id').value;
      if (!friendId) return;

      try {
        await API.user.sendFriendRequest(friendId);
        Toast.success("Đã gửi yêu cầu kết bạn!");
        document.getElementById('add-friend-id').value = '';
      } catch (err) {
        Toast.error(err.message || "Không thể gửi yêu cầu kết bạn.");
      }
    };

    // Status presence widget click toggle
    const statusWidget = document.getElementById('status-widget');
    const statusDot = document.getElementById('status-dot');
    const statusText = document.getElementById('status-text');
    
    const statuses = ['online', 'away', 'busy', 'offline'];
    statusWidget.onclick = async () => {
      let currentIdx = statuses.indexOf(statusText.innerText);
      let nextIdx = (currentIdx + 1) % statuses.length;
      let nextStatus = statuses[nextIdx];

      try {
        await API.user.updateStatus(nextStatus, '');
        statusText.innerText = nextStatus;
        statusDot.className = `presence-dot ${nextStatus}`;
        Toast.info(`Đã đổi trạng thái sang ${nextStatus.toUpperCase()}`);
      } catch (err) {
        Toast.error("Không thể cập nhật trạng thái.");
      }
    };

    // Avatar upload handler in settings
    const avatarInput = document.getElementById('avatar-file-input');
    const avatarPreview = document.getElementById('profile-avatar-preview');
    let uploadedAvatarUrl = user.avatar_url || '';

    avatarInput.onchange = async () => {
      const file = avatarInput.files[0];
      if (!file) return;

      const formData = new FormData();
      formData.append('file', file);
      formData.append('title', `avatar-${user.id}`);
      formData.append('artist', user.username);

      try {
        Toast.info("Đang tải ảnh đại diện lên MinIO...");
        // Tải ảnh lên bucket nhạc thông qua endpoint upload, lấy link public
        const trackData = await API.music.upload(formData);
        uploadedAvatarUrl = trackData.source_url;
        avatarPreview.innerHTML = `<img src="${uploadedAvatarUrl}" alt="Avatar">`;
        Toast.success("Ảnh đại diện đã được xử lý tải lên.");
      } catch (err) {
        Toast.error("Tải ảnh thất bại.");
      }
    };

    // Save profile handler
    document.getElementById('save-profile-btn').onclick = async () => {
      const disp = document.getElementById('profile-displayname').value;
      const bio = document.getElementById('profile-bio').value;

      try {
        const updated = await API.user.updateProfile(disp, bio, uploadedAvatarUrl);
        State.set('user', updated);
        
        // Update header & layout preview
        document.getElementById('slim-avatar-btn').innerHTML = updated.avatar_url ? `<img src="${updated.avatar_url}" alt="Avatar">` : `<span style="font-weight:600; font-size:12px;">${updated.display_name.slice(0, 2).toUpperCase()}</span>`;
        Modal.close('profile-settings-modal');
        Toast.success("Đã lưu cấu hình hồ sơ!");
      } catch (err) {
        Toast.error(err.message || "Lưu hồ sơ thất bại.");
      }
    };
  },

  async loadRooms() {
    const grid = document.getElementById('rooms-grid-container');
    try {
      const rooms = await API.room.list();
      
      if (rooms.length === 0) {
        grid.innerHTML = `
          <div style="grid-column: 1/-1; text-align:center; padding: 40px; color:var(--text-muted);">
            <p style="font-size:14px; margin-bottom:10px;">Chưa có phòng nào hoạt động công khai.</p>
            <span style="font-size:12.5px;">Hãy nhấn nút "+ Tạo phòng mới" ở trên để làm người mở đầu!</span>
          </div>
        `;
        return;
      }

      grid.innerHTML = rooms.map(room => `
        <div class="room-card" data-id="${room.id}">
          <div class="room-card-top">
            <h3>${room.name}</h3>
            <p>${room.description || 'Không có mô tả cho phòng này.'}</p>
          </div>
          <div class="room-card-bottom">
            <span class="room-members-count">
              <span style="font-size: 13px;">👥</span> ${room.active_members || 0}
            </span>
            ${room.current_song ? `
              <span class="room-playing-badge">
                <span style="font-size: 11px;">🎜</span> ${room.current_song}
              </span>
            ` : ''}
          </div>
        </div>
      `).join('');

      // Bind join room clicks
      grid.querySelectorAll('.room-card').forEach(card => {
        card.onclick = async () => {
          const roomId = card.getAttribute('data-id');
          await this.joinRoom(roomId);
        };
      });

    } catch (err) {
      grid.innerHTML = `<p style="color:var(--danger); grid-column:1/-1; text-align:center;">Lỗi tải danh sách phòng: ${err.message}</p>`;
    }
  },

  async joinRoom(roomId, presetPassword = '') {
    try {
      const joinData = await API.room.join(roomId, presetPassword);
      State.set('roomMemberRole', joinData.role);
      State.set('roomPermissions', joinData.permissions);
      
      const roomDetails = await API.room.get(roomId);
      State.set('activeRoom', roomDetails);
      
      this.stopPresenceHeartbeat();
      window.location.hash = `#/room/${roomId}`;
    } catch (err) {
      // Nếu phòng có password và chưa nhập mật khẩu
      if (err.message.includes("mật khẩu") || err.message.includes("PASSWORD")) {
        const pw = prompt("Hãy nhập mật khẩu phòng:");
        if (pw !== null) {
          await this.joinRoom(roomId, pw);
        }
      } else {
        Toast.error("Không thể tham gia phòng này: " + err.message);
      }
    }
  },

  async loadFriends() {
    const list = document.getElementById('friends-list-container');
    try {
      const friends = await API.user.getFriends();
      State.set('friends', friends);

      if (friends.length === 0) {
        list.innerHTML = `<p style="font-size:12.5px; color:var(--text-muted); text-align:center; margin-top:20px;">Bạn chưa kết bạn với ai.</p>`;
        return;
      }

      list.innerHTML = friends.map(friend => {
        // friendship chứa ID của bạn bè
        const profile = friend.friend_profile;
        const status = friend.presence?.status || 'offline';
        const customText = friend.presence?.custom_text || '';

        return `
          <div class="friend-item">
            <div class="friend-info">
              <div class="friend-avatar">
                ${profile.avatar_url ? `<img src="${profile.avatar_url}" alt="Avatar">` : `<span style="font-size:11px; font-weight:600;">${profile.display_name.slice(0, 2).toUpperCase()}</span>`}
                <div class="friend-presence-dot ${status}"></div>
              </div>
              <div class="friend-name-wrap">
                <span class="friend-name">${profile.display_name}</span>
                <span class="friend-status-text">${customText || status}</span>
              </div>
            </div>
          </div>
        `;
      }).join('');
    } catch (err) {
      list.innerHTML = `<p style="color:var(--danger); font-size:12px; text-align:center;">Lỗi tải bạn bè.</p>`;
    }
  },

  startPresenceHeartbeat() {
    this.statusTimer = setInterval(async () => {
      try {
        const status = document.getElementById('status-text').innerText;
        await API.user.updateStatus(status, 'Heartbeat active');
      } catch (e) {
        console.log("Heartbeat update failed:", e);
      }
    }, 45 * 1000); // 45s heartbeat to Redis
  },

  stopPresenceHeartbeat() {
    if (this.statusTimer) {
      clearInterval(this.statusTimer);
      this.statusTimer = null;
    }
  }
};
