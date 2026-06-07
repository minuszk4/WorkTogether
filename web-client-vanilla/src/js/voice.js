import { Room, RoomEvent } from 'livekit-client';
import { API } from './api.js';
import { State } from './state.js';
import { Toast } from './components/toast.js';

export class VoiceClient {
  constructor(roomId) {
    this.roomId = roomId;
    this.room = null;
    this.voiceGrid = null;
  }

  async connect(voiceGridEl) {
    this.voiceGrid = voiceGridEl;

    try {
      // 1. Lấy Token từ voice-service
      Toast.info("Đang xin cấp quyền Voice Channel...");
      const tokenData = await API.voice.getToken(this.roomId);
      const token = tokenData.token;

      // 2. Khởi tạo LiveKit Room
      this.room = new Room({
        adaptiveStream: true,
        dynacast: true,
      });

      // 3. Lắng nghe sự kiện LiveKit
      this.setupRoomListeners();

      // 4. Kết nối đến LiveKit Server SFU
      // LiveKit server chạy cổng 7880
      const sfuUrl = 'ws://localhost:7880';
      await this.room.connect(sfuUrl, token);
      
      State.set('voiceConnected', true);
      Toast.success("Đã kết nối vào kênh Voice thoại.");

      // 5. Bật mic và publish audio track lên phòng
      await this.room.localParticipant.setMicrophoneEnabled(true);
      
      // Render bản thân lên grid
      this.renderParticipantList();

    } catch (err) {
      console.log("Lỗi kết nối Voice Call:", err);
      Toast.error("Không thể kết nối Voice thoại: " + err.message);
      this.disconnect();
    }
  }

  setupRoomListeners() {
    if (!this.room) return;

    this.room
      .on(RoomEvent.ParticipantConnected, () => {
        this.renderParticipantList();
      })
      .on(RoomEvent.ParticipantDisconnected, () => {
        this.renderParticipantList();
      })
      .on(RoomEvent.TrackSubscribed, (track, publication, participant) => {
        // Track audio được tự động phát bởi LiveKit SDK, ta chỉ việc hiển thị UI
        this.renderParticipantList();
      })
      .on(RoomEvent.TrackUnsubscribed, () => {
        this.renderParticipantList();
      })
      .on(RoomEvent.ActiveSpeakersChanged, (speakers) => {
        this.highlightSpeakers(speakers);
      })
      .on(RoomEvent.Disconnected, () => {
        State.set('voiceConnected', false);
        this.renderParticipantList();
      });
  }

  renderParticipantList() {
    if (!this.voiceGrid) return;
    
    if (!this.room || this.room.state === 'disconnected') {
      this.voiceGrid.innerHTML = `
        <div style="font-size:12px; color:var(--text-muted); padding: 10px;">
          Chưa kết nối Voice Call. Nhấn nút thoại để vào.
        </div>
      `;
      return;
    }

    const participants = Array.from(this.room.participants.values());
    const local = this.room.localParticipant;
    
    // Thêm local participant lên đầu danh sách
    const allParticipants = [local, ...participants];

    this.voiceGrid.innerHTML = allParticipants.map(p => {
      const isLocal = p.sid === local.sid;
      const isMuted = !p.isMicrophoneEnabled;
      const name = p.identity || "Unknown User";

      return `
        <div class="voice-participant-box" id="voice-p-${p.sid}">
          <div class="voice-avatar">
            <span style="font-size: 10px; font-weight:600; display:flex; align-items:center; justify-content:center; height:100%; width:100%;">
              ${name.slice(0, 2).toUpperCase()}
            </span>
          </div>
          <span class="voice-name">${name} ${isLocal ? '(Bạn)' : ''}</span>
          ${isMuted ? `
            <span class="voice-indicator-icon" style="color:var(--danger)">🔇</span>
          ` : `
            <span class="voice-indicator-icon">🎤</span>
          `}
        </div>
      `;
    }).join('');
  }

  highlightSpeakers(speakers) {
    if (!this.voiceGrid) return;

    // Bỏ highlight cũ
    this.voiceGrid.querySelectorAll('.voice-participant-box').forEach(box => {
      box.classList.remove('speaking');
    });

    // Thêm viền xanh lá nhấp nháy cho người đang phát biểu
    speakers.forEach(speaker => {
      const box = document.getElementById(`voice-p-${speaker.sid}`);
      if (box) {
        box.classList.add('speaking');
      }
    });
  }

  async setMute(muted) {
    if (!this.room || !this.room.localParticipant) return;
    
    try {
      await this.room.localParticipant.setMicrophoneEnabled(!muted);
      this.renderParticipantList();
      Toast.info(muted ? "Đã tắt mic." : "Đã mở mic.");
    } catch (e) {
      console.log("Lỗi thay đổi trạng thái mic:", e);
    }
  }

  disconnect() {
    if (this.room) {
      this.room.disconnect();
      this.room = null;
    }
    State.set('voiceConnected', false);
    if (this.voiceGrid) {
      this.voiceGrid.innerHTML = `
        <div style="font-size:12px; color:var(--text-muted); padding: 10px;">
          Chưa kết nối Voice Call. Nhấn nút thoại để vào.
        </div>
      `;
    }
  }
}
