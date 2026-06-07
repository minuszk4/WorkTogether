import { API } from './api.js';
import { State } from './state.js';
import { Toast } from './components/toast.js';
import { ChatClient } from './chat.js';
import { PlaybackClient } from './playback.js';
import { VoiceClient } from './voice.js';

export const RoomView = {
  container: null,
  roomId: null,
  chatClient: null,
  playbackClient: null,
  voiceClient: null,
  
  activePlaylistId: null,
  isMuted: false,

  init(container) {
    this.container = container;
  },

  async render(roomId) {
    this.roomId = roomId;
    const user = State.get('user');
    const token = State.get('accessToken');
    const room = State.get('activeRoom');

    if (!user || !room) {
      window.location.hash = '#/dashboard';
      return;
    }

    this.container.innerHTML = `
      <div class="room-layout">
        <!-- Slim Left Navigation (64px) -->
        <div class="dashboard-sidebar-slim">
          <div class="flex flex-col gap-4 items-center">
            <div class="slim-nav-item" id="room-back-btn" title="Quay về Lobby">
              <span style="font-weight: 800; font-size: 14px;">➔</span>
            </div>
            
            <!-- Voice mic control -->
            <div class="slim-nav-item" id="room-mic-btn" title="Mute/Unmute Microphone">
              <span style="font-size: 16px;" id="room-mic-icon">🎤</span>
            </div>
          </div>
          
          <div class="flex flex-col gap-4 items-center">
            <div class="slim-avatar" title="Hồ sơ cá nhân">
              ${user.avatar_url ? `<img src="${user.avatar_url}" alt="Avatar">` : user.display_name.slice(0, 2).toUpperCase()}
            </div>
          </div>
        </div>

        <!-- Column 1: Info & Chat Panel (280px) -->
        <div class="room-chat-panel">
          <div class="room-info-header">
            <h2>${room.name}</h2>
            <p>${room.description || 'Không có mô tả.'}</p>
          </div>

          <!-- Pinned Messages Banner -->
          <div class="pinned-msg-banner" id="pinned-banner-container" style="display: none;">
            <span style="font-size:11px;">📌 Ghim: <span id="pinned-msg-text">...</span></span>
          </div>

          <!-- Chat messages area -->
          <div class="chat-messages" id="chat-messages-container">
            <p style="font-size:12px; color:var(--text-muted); text-align:center; padding-top:20px;">Đang kết nối chat...</p>
          </div>

          <!-- Chat input -->
          <div class="chat-input-area">
            <form id="chat-send-form" class="chat-input-wrapper">
              <input type="text" id="chat-input-field" class="chat-text-input" placeholder="Nhập tin nhắn..." autocomplete="off">
              <button type="submit" style="display:none;"></button>
            </form>
          </div>
        </div>

        <!-- Column 2: Central Workspace (Media & Voice Grid) -->
        <div class="room-workspace-main">
          <!-- Voice Grid -->
          <div class="voice-participants-grid" id="voice-grid-container">
            <div style="font-size:12px; color:var(--text-muted); padding: 10px;">
              Chưa kết nối Voice Call. Hãy mở mic để tham gia.
            </div>
          </div>

          <!-- YouTube viewport -->
          <div class="media-viewport">
            <div class="youtube-player-frame">
              <div id="youtube-player-element"></div>
            </div>
          </div>

          <!-- Persistent Bottom Player Bar -->
          <div class="music-player-bar">
            <!-- Track Info -->
            <div class="player-track-info">
              <div class="player-track-thumb" id="player-track-thumb">
                <span style="font-size:16px;">🎜</span>
              </div>
              <div class="player-track-details">
                <span class="player-track-title" id="player-track-title">Chưa phát bài hát nào</span>
                <span class="player-track-artist" id="player-track-artist">Hàng đợi đang trống</span>
              </div>
            </div>

            <!-- Controls Center -->
            <div class="player-controls-center">
              <div class="player-buttons">
                <button class="player-btn btn-play-pause" id="play-pause-btn">
                  <span style="font-size: 16px;">▶</span>
                </button>
              </div>
              
              <!-- Progress Scrubber -->
              <div class="player-progress-container">
                <span class="player-time" id="player-time-current">00:00</span>
                <div class="player-scrubber-bar" id="player-scrubber">
                  <div class="player-progress-fill" id="player-fill"></div>
                  <div class="player-scrubber-handle" id="player-handle" style="left: 0%;"></div>
                </div>
                <span class="player-time" id="player-time-total">00:00</span>
              </div>
            </div>

            <!-- Volume / Info Right -->
            <div class="player-volume-right">
              <span style="font-size: 14px;">🔊</span>
              <div class="player-volume-bar" id="volume-bar">
                <div class="player-volume-fill" id="volume-fill"></div>
              </div>
            </div>
          </div>
        </div>

        <!-- Column 3: Playback Queue & Members (300px) -->
        <div class="room-right-panel">
          <div class="right-panel-tabs">
            <button class="panel-tab-btn active" data-tab="tab-queue">Hàng đợi</button>
            <button class="panel-tab-btn" data-tab="tab-members">Thành viên</button>
          </div>

          <div class="right-panel-content">
            <!-- Queue Tab -->
            <div class="tab-pane active" id="tab-queue">
              <div style="display:flex; flex-direction:column; gap:12px;">
                <div class="flex gap-2">
                  <input type="text" id="add-queue-url" placeholder="URL YouTube..." style="font-size:12.5px; padding:6px 10px;">
                  <button class="btn btn-primary" id="add-queue-btn" style="padding:6px 10px; font-size:12px;">Thêm</button>
                </div>
                
                <div class="queue-list" id="queue-list-container">
                  <p style="font-size:12px; color:var(--text-muted); text-align:center; margin-top:10px;">Hàng đợi đang trống.</p>
                </div>
              </div>
            </div>

            <!-- Members Tab -->
            <div class="tab-pane" id="tab-members">
              <div class="queue-list" id="members-list-container">
                <p style="font-size:12px; color:var(--text-muted); text-align:center;">Đang tải...</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    `;

    // 3. Khởi tạo & Kết nối các WebSockets
    this.chatClient = new ChatClient(roomId, token);
    this.chatClient.connect(document.getElementById('chat-messages-container'));

    this.playbackClient = new PlaybackClient(roomId, token);
    this.playbackClient.connect('youtube-player-element');

    this.voiceClient = new VoiceClient(roomId);

    // Bind events
    this.bindEvents();
    
    // Load Playlist & Queue
    await this.setupPlaylist();
  }

  bindEvents() {
    // Back to dashboard
    document.getElementById('room-back-btn').onclick = async () => {
      this.disconnectAll();
      window.location.hash = '#/dashboard';
    };

    // Chat submit
    const chatForm = document.getElementById('chat-send-form');
    chatForm.onsubmit = (e) => {
      e.preventDefault();
      const input = document.getElementById('chat-input-field');
      const text = input.value.trim();
      if (!text) return;

      this.chatClient.sendMessage(text);
      input.value = '';
    };

    // Microphone toggle voice call join/leave
    const micBtn = document.getElementById('room-mic-btn');
    const micIcon = document.getElementById('room-mic-icon');
    
    micBtn.onclick = async () => {
      const isVoiceConnected = State.get('voiceConnected');

      if (!isVoiceConnected) {
        // Vào Voice Call
        micIcon.innerText = "🎤";
        micBtn.classList.add('active');
        await this.voiceClient.connect(document.getElementById('voice-grid-container'));
      } else {
        // Tắt microphone (Mute) / Thoát Voice Call
        const action = confirm("Bạn muốn Mute Mic hay Thoát khỏi Voice Call?\nNhấn OK để Mute, CANCEL để Thoát Kênh.");
        if (action) {
          this.isMuted = !this.isMuted;
          await this.voiceClient.setMute(this.isMuted);
          micIcon.innerText = this.isMuted ? "🔇" : "🎤";
          if (this.isMuted) micBtn.classList.remove('active');
          else micBtn.classList.add('active');
        } else {
          this.voiceClient.disconnect();
          micIcon.innerText = "🎤";
          micBtn.classList.remove('active');
          this.isMuted = false;
        }
      }
    };

    // Right panel tab switching
    const tabBtns = document.querySelectorAll('.panel-tab-btn');
    const tabPanes = document.querySelectorAll('.tab-pane');

    tabBtns.forEach(btn => {
      btn.onclick = () => {
        tabBtns.forEach(b => b.classList.remove('active'));
        tabPanes.forEach(p => p.classList.remove('active'));

        btn.classList.add('active');
        const tabId = btn.getAttribute('data-tab');
        document.getElementById(tabId).classList.add('active');
        
        if (tabId === 'tab-members') {
          this.loadRoomMembers();
        }
      };
    });

    // Volume bar handler
    const volBar = document.getElementById('volume-bar');
    const volFill = document.getElementById('volume-fill');
    if (volBar) {
      volBar.onclick = (e) => {
        const rect = volBar.getBoundingClientRect();
        const pct = (e.clientX - rect.left) / rect.width;
        const vol = Math.floor(pct * 100);
        
        if (volFill) volFill.style.width = `${vol}%`;
        
        // Cập nhật volume YouTube Player
        if (this.playbackClient.ytPlayer && this.playbackClient.ytPlayer.setVolume) {
          this.playbackClient.ytPlayer.setVolume(vol);
          Toast.info(`Âm lượng: ${vol}%`);
        }
      };
    }

    // Add Queue Music Link submit
    document.getElementById('add-queue-btn').onclick = async () => {
      const urlInput = document.getElementById('add-queue-url');
      const url = urlInput.value.trim();
      if (!url) return;

      try {
        Toast.info("Đang trích xuất video từ YouTube...");
        const track = await API.music.extract(url);
        
        Toast.info("Thêm bài hát vào danh sách nhạc...");
        await API.playlist.addTrack(this.activePlaylistId, track);
        
        Toast.success(`Đã thêm bài hát "${track.title}" vào Queue.`);
        urlInput.value = '';
        
        // Reload queue list
        await this.loadQueueTracks();
      } catch (err) {
        Toast.error("Thêm nhạc thất bại: " + err.message);
      }
    };
  },

  async setupPlaylist() {
    try {
      const playlists = await API.playlist.getRoomPlaylists(this.roomId);
      
      if (playlists.length === 0) {
        // Tự khởi tạo playlist mặc định nếu chưa có
        const newPlaylist = await API.playlist.create("Default Playlist", this.roomId);
        this.activePlaylistId = newPlaylist.id;
      } else {
        this.activePlaylistId = playlists[0].id;
      }

      await this.loadQueueTracks();
    } catch (err) {
      console.log("Lỗi cài đặt Playlist phòng:", err);
    }
  },

  async loadQueueTracks() {
    const list = document.getElementById('queue-list-container');
    if (!this.activePlaylistId) return;

    try {
      const tracks = await API.playlist.getTracks(this.activePlaylistId);
      
      if (tracks.length === 0) {
        list.innerHTML = `<p style="font-size:12px; color:var(--text-muted); text-align:center; margin-top:10px;">Hàng đợi đang trống.</p>`;
        return;
      }

      list.innerHTML = tracks.map(track => {
        const isUpvoted = false; // logic lưu local list/votes if needed
        return `
          <div class="queue-item" data-id="${track.id}">
            <div class="queue-item-info">
              <span class="queue-item-title" title="${track.title}">${track.title}</span>
              <span class="queue-item-artist">${track.artist || 'Unknown'}</span>
            </div>
            <div class="queue-item-actions">
              <span class="vote-badge ${isUpvoted ? 'upvoted' : ''}" data-item-id="${track.id}">
                ▲ ${track.votes || 0}
              </span>
              <button class="friend-btn-icon remove-track-btn" style="color:var(--danger);" data-item-id="${track.id}" title="Xóa">×</button>
            </div>
          </div>
        `;
      }).join('');

      // Bind Upvote click
      list.querySelectorAll('.vote-badge').forEach(badge => {
        badge.onclick = async (e) => {
          e.stopPropagation();
          const itemID = badge.getAttribute('data-item-id');
          try {
            // Toggle vote up
            await API.playlist.voteTrack(itemID, "up");
            await this.loadQueueTracks();
          } catch (err) {
            Toast.error("Không thể bình chọn.");
          }
        };
      });

      // Bind Remove click
      list.querySelectorAll('.remove-track-btn').forEach(btn => {
        btn.onclick = async (e) => {
          e.stopPropagation();
          const itemID = btn.getAttribute('data-item-id');
          try {
            await API.playlist.removeTrack(this.activePlaylistId, itemID);
            Toast.success("Đã xóa khỏi hàng đợi.");
            await this.loadQueueTracks();
          } catch (err) {
            Toast.error("Xóa thất bại.");
          }
        };
      });

      // Bind Click to Play immediately
      list.querySelectorAll('.queue-item').forEach(item => {
        item.onclick = () => {
          const itemID = item.getAttribute('data-id');
          const track = tracks.find(t => t.id === itemID);
          if (track) {
            // Gửi lệnh điều khiển Playback để bắt đầu bài hát này
            this.playbackClient.sendControlCommand("play", 0);
            
            // Cập nhật thông tin bài hát trong player client của mình trước
            const ctrlReq = {
              Action: "play",
              TrackID: track.track_id,
              PositionMS: 0,
              Title: track.title,
              Artist: track.artist,
              ThumbnailURL: track.thumbnail,
              DurationMS: track.duration_ms,
              SourceURL: track.source_url
            };
            this.playbackClient.sendControlCommand("play", 0);
          }
        };
      });

    } catch (err) {
      list.innerHTML = `<p style="font-size:11px; color:var(--danger); text-align:center;">Lỗi tải queue.</p>`;
    }
  },

  async loadRoomMembers() {
    const list = document.getElementById('members-list-container');
    try {
      // Vì mockup demo và gRPC room-members được kết nối nội bộ
      // Ta có thể gọi thông tin room từ State
      const room = State.get('activeRoom');
      if (!room) return;

      // Giả lập danh sách thành viên dựa trên vai trò
      list.innerHTML = `
        <div class="room-member-row">
          <span style="font-size:13px; font-weight:500;">👤 Admin Owner</span>
          <span class="member-role-badge owner">Owner</span>
        </div>
        <div class="room-member-row">
          <span style="font-size:13px; font-weight:500;">👤 Coder_A</span>
          <span class="member-role-badge moderator">Mod</span>
        </div>
        <div class="room-member-row">
          <span style="font-size:13px; font-weight:500;">👤 Guest_B</span>
          <span class="member-role-badge">Member</span>
        </div>
      `;
    } catch (err) {
      list.innerHTML = `<p style="font-size:11px; color:var(--text-muted); text-align:center;">Lỗi tải thành viên.</p>`;
    }
  },

  disconnectAll() {
    if (this.chatClient) this.chatClient.disconnect();
    if (this.playbackClient) this.playbackClient.disconnect();
    if (this.voiceClient) this.voiceClient.disconnect();
    State.set('activeRoom', null);
  }
};
