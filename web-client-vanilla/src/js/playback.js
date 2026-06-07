import { Toast } from './components/toast.js';

export class PlaybackClient {
  constructor(roomId, token) {
    this.roomId = roomId;
    this.token = token;
    this.socket = null;
    this.ytPlayer = null;
    
    // NTP variables
    this.clockOffset = 0; // theta
    this.rtt = 0;
    this.pingInterval = null;
    
    // Scrubber elements
    this.progressBar = null;
    this.progressFill = null;
    this.currentTimeText = null;
    this.totalTimeText = null;
    
    this.progressTimer = null;
    
    // Current playback state
    this.currentState = "stopped";
    this.currentTrack = null;
    this.lastServerPosition = 0;
    this.lastUpdatedAt = 0; // T3 from server
    
    // Lock variable to prevent sync loop (e.g. user seeks -> triggers WS -> WS seeks -> triggers seek loop)
    this.isSyncingLocal = false;
  }

  connect(ytPlayerContainerId) {
    // 1. Khởi chạy Youtube IFrame Player
    this.initYoutubePlayer(ytPlayerContainerId);

    // 2. Kết nối WebSocket Playback
    const wsUrl = `ws://localhost:8080/api/v1/rooms/${this.roomId}/playback/ws?token=${this.token}`;
    this.socket = new WebSocket(wsUrl);

    this.socket.onopen = () => {
      console.log("[Playback WS] Kết nối thành công.");
      // Bắt đầu đo NTP-like clock offset ngay lập tức
      this.sendNTPPing();
      this.pingInterval = setInterval(() => this.sendNTPPing(), 10000); // Mỗi 10s ping một lần
    };

    this.socket.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    this.socket.onerror = (err) => {
      console.log("[Playback WS] Lỗi kết nối:", err);
    };

    this.socket.onclose = () => {
      console.log("[Playback WS] Ngắt kết nối.");
      if (this.pingInterval) clearInterval(this.pingInterval);
      if (this.progressTimer) clearInterval(this.progressTimer);
    };

    this.initUIElements();
  }

  initUIElements() {
    this.progressBar = document.getElementById('player-scrubber');
    this.progressFill = document.getElementById('player-fill');
    this.currentTimeText = document.getElementById('player-time-current');
    this.totalTimeText = document.getElementById('player-time-total');

    // Scrubber click seek handler
    if (this.progressBar) {
      this.progressBar.onclick = (e) => {
        if (!this.currentTrack) return;
        const rect = this.progressBar.getBoundingClientRect();
        const pct = (e.clientX - rect.left) / rect.width;
        const seekMS = Math.floor(pct * this.currentTrack.duration_ms);
        
        this.sendControlCommand("seek", seekMS);
      };
    }

    // Play/Pause buttons click
    const playPauseBtn = document.getElementById('play-pause-btn');
    if (playPauseBtn) {
      playPauseBtn.onclick = () => {
        if (!this.currentTrack) return;
        const action = this.currentState === "playing" ? "pause" : "play";
        const currentPos = this.getLocalProgress();
        this.sendControlCommand(action, currentPos);
      };
    }
  }

  initYoutubePlayer(containerId) {
    const triggerYTInit = () => {
      this.ytPlayer = new YT.Player(containerId, {
        height: '100%',
        width: '100%',
        videoId: '', // start empty
        playerVars: {
          autoplay: 0,
          controls: 0, // Dùng controls tùy biến ở UI
          disablekb: 1,
          fs: 0,
          rel: 0
        },
        events: {
          onReady: () => {
            console.log("[YouTube Player] Sẵn sàng.");
            this.syncPlayerWithServerState();
          },
          onStateChange: (event) => {
            this.handlePlayerStateChange(event.data);
          }
        }
      });
    };

    if (window.YT && window.YT.Player) {
      triggerYTInit();
    } else {
      // Đợi Script YouTube load
      window.onYouTubeIframeAPIReady = triggerYTInit;
    }
  }

  // NTP Ping
  sendNTPPing() {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;

    const payload = {
      event: "sync:ping",
      payload: {
        t1: Date.now() // client timestamp
      }
    };
    this.socket.send(JSON.stringify(payload));
  }

  handleMessage(dataStr) {
    try {
      const msg = JSON.parse(dataStr);

      switch (msg.event) {
        case "sync:pong":
          this.calculateClockOffset(msg.payload);
          break;
        case "playback:sync":
          this.handlePlaybackSync(msg.payload);
          break;
      }
    } catch (e) {
      console.log("Lỗi parse playback WS data:", e);
    }
  }

  calculateClockOffset(payload) {
    const t4 = Date.now(); // Client nhận
    const t1 = payload.t1;
    const t2 = payload.t2; // Server nhận
    const t3 = payload.t3; // Server gửi

    // Công thức RTT & Clock Offset
    this.rtt = (t4 - t1) - (t3 - t2);
    this.clockOffset = ((t2 - t1) + (t3 - t4)) / 2;

    console.log(`[NTP Sync] RTT: ${this.rtt}ms, Clock Offset: ${this.clockOffset}ms`);
  }

  handlePlaybackSync(payload) {
    this.currentState = payload.state;
    this.lastServerPosition = payload.position_ms;
    this.lastUpdatedAt = payload.updated_at;

    // Cập nhật thông tin bài hát hiện tại
    const titleEl = document.getElementById('player-track-title');
    const artistEl = document.getElementById('player-track-artist');
    const thumbEl = document.getElementById('player-track-thumb');
    
    if (payload.current_track_id) {
      this.currentTrack = {
        id: payload.current_track_id,
        title: payload.title || "Unknown Title",
        artist: payload.artist || "Unknown Artist",
        thumbnail_url: payload.thumbnail_url || "",
        duration_ms: payload.duration_ms,
        source_url: payload.source_url
      };

      if (titleEl) titleEl.innerText = this.currentTrack.title;
      if (artistEl) artistEl.innerText = this.currentTrack.artist;
      if (thumbEl && this.currentTrack.thumbnail_url) {
        thumbEl.innerHTML = `<img src="${this.currentTrack.thumbnail_url}" alt="Thumbnail">`;
      }
      if (this.totalTimeText) {
        this.totalTimeText.innerText = this.formatTime(this.currentTrack.duration_ms);
      }
    } else {
      this.currentTrack = null;
      if (titleEl) titleEl.innerText = "Chưa phát bài hát nào";
      if (artistEl) artistEl.innerText = "Hàng đợi đang trống";
      if (thumbEl) thumbEl.innerHTML = `<span style="font-size:16px;">🎜</span>`;
      if (this.totalTimeText) this.totalTimeText.innerText = "00:00";
    }

    // Cập nhật nút Play/Pause UI
    const playPauseBtn = document.getElementById('play-pause-btn');
    if (playPauseBtn) {
      playPauseBtn.innerHTML = this.currentState === "playing" ? '<span style="font-size:16px;">⏸</span>' : '<span style="font-size:16px;">▶</span>';
    }

    // Đồng bộ hóa trạng thái Youtube Player
    this.syncPlayerWithServerState();
    
    // Khởi động render progress bar tại local
    this.startProgressTimer();
  }

  syncPlayerWithServerState() {
    if (!this.ytPlayer || !this.ytPlayer.cueVideoById) return;
    if (!this.currentTrack) {
      this.ytPlayer.stopVideo();
      return;
    }

    // Trích xuất videoId từ source_url
    const videoId = this.extractYoutubeId(this.currentTrack.source_url);
    if (!videoId) return;

    const currentYTVideo = this.ytPlayer.getVideoData ? this.ytPlayer.getVideoData().video_id : '';
    
    // 1. Tính toán vị trí đồng bộ chuẩn hóa bằng NTP offset
    const serverTimeEst = Date.now() + this.clockOffset;
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === "playing") {
      targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    }
    const targetPosSec = targetPosMS / 1000;

    // Set lock
    this.isSyncingLocal = true;

    // Nạp video nếu khác
    if (currentYTVideo !== videoId) {
      this.ytPlayer.cueVideoById({
        videoId: videoId,
        startSeconds: targetPosSec > 0 ? targetPosSec : 0
      });
    }

    // Điều khiển play/pause/seek khớp server
    setTimeout(() => {
      const playerState = this.ytPlayer.getPlayerState ? this.ytPlayer.getPlayerState() : -1;
      
      if (this.currentState === "playing") {
        const localTimeSec = this.ytPlayer.getCurrentTime ? this.ytPlayer.getCurrentTime() : 0;
        const drift = Math.abs(localTimeSec - targetPosSec);

        // Nếu lệch quá 1.5 giây, nhảy seek
        if (drift > 1.5) {
          this.ytPlayer.seekTo(targetPosSec, true);
        }
        
        // Play nếu chưa chạy
        if (playerState !== YT.PlayerState.PLAYING) {
          this.ytPlayer.playVideo();
        }
      } else if (this.currentState === "paused") {
        this.ytPlayer.seekTo(targetPosSec, true);
        this.ytPlayer.pauseVideo();
      } else {
        this.ytPlayer.stopVideo();
      }
      
      // Mở khóa sau khi hoàn tất sync
      setTimeout(() => {
        this.isSyncingLocal = false;
      }, 500);

    }, currentYTVideo !== videoId ? 800 : 50);
  }

  handlePlayerStateChange(state) {
    if (this.isSyncingLocal || !this.currentTrack) return;

    // Người dùng bấm play/pause thủ công trên frame video
    let action = "";
    const currentPosMS = Math.floor((this.ytPlayer.getCurrentTime() || 0) * 1000);

    if (state === YT.PlayerState.PLAYING && this.currentState !== "playing") {
      action = "play";
    } else if (state === YT.PlayerState.PAUSED && this.currentState === "playing") {
      action = "pause";
    }

    if (action !== "") {
      this.sendControlCommand(action, currentPosMS);
    }
  }

  sendControlCommand(action, positionMS) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN || !this.currentTrack) return;

    const payload = {
      event: "playback:control",
      payload: {
        action: action,
        track_id: this.currentTrack.id,
        position_ms: positionMS,
        title: this.currentTrack.title,
        artist: this.currentTrack.artist,
        thumbnail_url: this.currentTrack.thumbnail_url,
        duration_ms: this.currentTrack.duration_ms,
        source_url: this.currentTrack.source_url
      }
    };
    
    this.socket.send(JSON.stringify(payload));
  }

  getLocalProgress() {
    if (this.currentState !== "playing" || this.lastUpdatedAt === 0) {
      return this.lastServerPosition;
    }
    const serverTimeEst = Date.now() + this.clockOffset;
    const elapsed = serverTimeEst - this.lastUpdatedAt;
    let pos = this.lastServerPosition + elapsed;
    
    if (this.currentTrack && pos > this.currentTrack.duration_ms) {
      pos = this.currentTrack.duration_ms;
    }
    return pos;
  }

  startProgressTimer() {
    if (this.progressTimer) clearInterval(this.progressTimer);

    this.progressTimer = setInterval(() => {
      if (!this.currentTrack) {
        if (this.progressFill) this.progressFill.style.width = '0%';
        if (this.currentTimeText) this.currentTimeText.innerText = "00:00";
        return;
      }

      const currentMS = this.getLocalProgress();
      const pct = (currentMS / this.currentTrack.duration_ms) * 100;

      if (this.progressFill) this.progressFill.style.width = `${pct}%`;
      if (this.currentTimeText) {
        this.currentTimeText.innerText = this.formatTime(currentMS);
      }
    }, 200); // render progress bar mượt mà sau mỗi 200ms
  }

  formatTime(ms) {
    const secTotal = Math.floor(ms / 1000);
    const min = Math.floor(secTotal / 60);
    const sec = secTotal % 60;
    return `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
  }

  extractYoutubeId(urlStr) {
    const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
    const match = urlStr.match(regExp);
    return (match && match[2].length === 11) ? match[2] : null;
  }

  disconnect() {
    if (this.socket) this.socket.close();
    if (this.pingInterval) clearInterval(this.pingInterval);
    if (this.progressTimer) clearInterval(this.progressTimer);
    if (this.ytPlayer && this.ytPlayer.destroy) this.ytPlayer.destroy();
  }
}
