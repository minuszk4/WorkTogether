import { State } from './state.js';
import { Toast } from './components/toast.js';

export class ChatClient {
  constructor(roomId, token) {
    this.roomId = roomId;
    this.token = token;
    this.socket = null;
    this.container = null;
  }

  connect(containerEl) {
    this.container = containerEl;
    const wsUrl = `ws://localhost:8080/api/v1/rooms/${this.roomId}/chat/ws?token=${this.token}`;

    this.socket = new WebSocket(wsUrl);

    this.socket.onopen = () => {
      console.log("[Chat WS] Kết nối thành công.");
      this.container.innerHTML = `<p style="font-size:11.5px; color:var(--text-muted); text-align:center; padding: 10px 0;">Đã kết nối vào phòng chat.</p>`;
    };

    this.socket.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.socket.onerror = (err) => {
      console.log("[Chat WS] Lỗi kết nối:", err);
      Toast.error("Mất kết nối Chat. Đang kết nối lại...");
    };

    this.socket.onclose = () => {
      console.log("[Chat WS] Đã ngắt kết nối.");
    };
  }

  handleMessage(dataStr) {
    try {
      const msg = JSON.parse(dataStr);
      
      switch (msg.event) {
        case "chat:message_received":
          this.renderNewMessage(msg.payload);
          break;
        case "chat:message_deleted":
          this.removeMessage(msg.payload.id);
          break;
        case "chat:pin_updated":
          this.updatePinnedBanner(msg.payload);
          break;
        default:
          console.log("[Chat WS] Sự kiện chưa đăng ký:", msg.event);
      }
    } catch (e) {
      console.log("Lỗi xử lý WS data:", e);
    }
  }

  renderNewMessage(payload) {
    const isAtBottom = this.container.scrollHeight - this.container.scrollTop <= this.container.clientHeight + 100;

    const msgCard = document.createElement('div');
    msgCard.className = 'chat-msg-card';
    msgCard.id = `msg-${payload.id}`;

    const sender = payload.sender;
    const avatar = sender.avatar_url;
    const name = sender.display_name || sender.username;
    
    // Parse time
    const timeVal = new Date(payload.created_at);
    const timeStr = `${String(timeVal.getHours()).padStart(2, '0')}:${String(timeVal.getMinutes()).padStart(2, '0')}`;

    msgCard.innerHTML = `
      <div class="chat-msg-avatar">
        ${avatar ? `<img src="${avatar}" alt="Avatar">` : `<span style="font-size:10px; font-weight:600; display:flex; align-items:center; justify-content:center; height:100%;">${name.slice(0, 2).toUpperCase()}</span>`}
      </div>
      <div class="chat-msg-content-wrap">
        <div class="chat-msg-header">
          <span class="chat-msg-sender">${name}</span>
          <span class="chat-msg-time">${timeStr}</span>
        </div>
        <div class="chat-msg-bubble">${payload.content}</div>
      </div>
    `;

    this.container.appendChild(msgCard);

    // Auto-scroll if user is looking at bottom
    if (isAtBottom) {
      this.container.scrollTop = this.container.scrollHeight;
    }
  }

  removeMessage(messageId) {
    const el = document.getElementById(`msg-${messageId}`);
    if (el) {
      const bubble = el.querySelector('.chat-msg-bubble');
      if (bubble) {
        bubble.innerHTML = `<span style="color:var(--text-muted); font-style:italic;">Tin nhắn đã bị thu hồi.</span>`;
      }
    }
  }

  updatePinnedBanner(payload) {
    const banner = document.getElementById('pinned-banner-container');
    if (!banner) return;

    if (payload && payload.content) {
      banner.style.display = 'flex';
      banner.querySelector('#pinned-msg-text').innerText = payload.content;
    } else {
      banner.style.display = 'none';
    }
  }

  sendMessage(content, replyToId = null) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      Toast.error("Không thể gửi tin nhắn. Kênh chat chưa sẵn sàng.");
      return;
    }

    const payload = {
      event: "chat:send_message",
      room_id: this.roomId,
      payload: {
        content: content,
        reply_to_id: replyToId
      }
    };

    this.socket.send(JSON.stringify(payload));
  }

  disconnect() {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }
}
