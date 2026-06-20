# SyncSpace — Room Redesign (Cool Ocean, Player-Centric)

**Ngày:** 2026-06-20
**Branch:** FE-Redesign
**Scope:** Front-end redesign — Phase 1: Room (lõi sản phẩm)
**Target app:** `web-client` (Angular 19)
**Approach:** B — Rebuild Room component từ đầu, port logic từ services hiện có

---

## 1. Bối cảnh & Mục tiêu

SyncSpace là nền tảng cộng tác thời gian thực: nghe nhạc cùng nhau (đồng bộ playback), chat, voice/video call, screen share, học nhóm. FE chính là `web-client` (Angular 19) với 3 màn hình: Auth, Dashboard, Room.

Room hiện tại dùng theme "dark-tech electric violet" với layout **4 cột đều** (sidebar 64px | chat 272px | workspace | queue 296px). Player bị "kẹt" giữa các panel, không nổi bật; design thiếu nhất quán; brand direction tối, ít "alive".

### Mục tiêu đợt redesign (toàn diện, 4 trục)
1. **Nâng tầm visual** — cao cấp, hiện đại, ấn tượng hơn.
2. **Cải thiện UX/flow** — player làm trung tâm, panel hợp lý hơn.
3. **Đổi mới brand direction** — chuyển từ violet sang "Fresh & Vibrant / Cool Ocean".
4. **Standardize & cleanup** — design system nhất quán, single source of truth, a11y.

### Phạm vi Phase 1
- **Chỉ Room** (lõi, nơi user ở lâu nhất).
- Auth & Dashboard để Phase sau (sẽ tái sử dụng design system & shared primitives từ Phase 1).
- Component Room cũ được **giữ lại làm fallback** trong quá trình chuyển đổi (không xóa ngay) cho đến khi Room mới đạt feature-parity và được verify.

---

## 2. Design System — Cool Ocean (Dark)

Đổi hoàn toàn identity từ "electric violet" sang "Cool Ocean" trên base **dark navy sâu** (không phải pure black).

### 2.1 Palette

```
Background (deep ocean navy):
  --bg-base:       #0a1628   (deep ocean navy, base)
  --bg-surface:    #0f2240   (card/panel)
  --bg-elevated:   #163355   (hover/modal)
  --bg-glass:      rgba(15, 34, 64, 0.6)   (frosted panel — chat/queue overlay)

Accent (ocean gradients — core identity, teal-led):
  --ocean-teal:    #2dd4bf   (primary accent)
  --ocean-sky:     #38bdf8   (secondary accent)
  --ocean-indigo:  #6366f1   (deep accent)
  --accent-primary:   #2dd4bf
  --accent-hover:     #34e0cc
  --accent-gradient:  linear-gradient(135deg, #2dd4bf 0%, #38bdf8 50%, #6366f1 100%)
  --accent-glow:      0 0 20px rgba(45, 212, 191, 0.18)
  --accent-glow-color: rgba(45, 212, 191, 0.18)

Borders:
  --border-color:  #1b3a5c
  --border-focus:  #2dd4bf

Text:
  --text-primary:   #e6f1ff   (cool white)
  --text-secondary: #8aa4c8   (cool grey-blue)
  --text-muted:     #4a6080

Semantic (điều chỉnh hue cho hợp palette cool):
  --success: #2dd4bf   (dùng teal làm "alive/ok")
  --warning: #fbbf24
  --danger:  #f87171   (warm red nhẹ, contrast trên dark navy)
```

### 2.2 Typography
- Font: **Outfit** (sans, giữ) + **JetBrains Mono** cho metadata/timestamp.
- Thêm **typography scale** chuẩn trong tokens: `--text-display` / `lg` / `md` / `sm` / `xs` + line-height tương ứng.

### 2.3 Radius & Motion
- Radius tăng cho cảm giác "fresh": `--radius-lg: 14px`, `--radius-xl: 22px`, pill buttons `--radius-full`.
- Motion: giữ `cubic-bezier(0.16, 1, 0.3, 1)`. Thêm biến `--transition-panel: 280ms`.

### 2.4 Shadows (bg-hue tinted, không pure black)
```
  --shadow-sm: 0 1px 3px rgba(4, 10, 22, 0.5)
  --shadow-md: 0 4px 12px rgba(4, 10, 22, 0.6)
  --shadow-lg: 0 12px 32px rgba(4, 10, 22, 0.7), 0 0 0 1px rgba(45, 212, 191, 0.04)
  --shadow-accent: 0 0 20px rgba(45, 212, 191, 0.2), 0 4px 12px rgba(4, 10, 22, 0.6)
```

### 2.5 Migration tokens
- `--accent-alpha` (legacy alias) sẽ bị xóa sau khi toàn bộ component migrate sang token mới.
- Trong quá trình rebuild, giữ tạm alias map sang teal để các component chưa rebuild không vỡ.

---

## 3. Layout Room — Player-Centric

### 3.1 Nguyên tắc
Player (music) là trung tâm, luôn có mặt. Voice/screenshare hiển thị adaptive. Chat & queue là panel phụ có thể toggle, không chiếm chỗ mặc định khi không cần.

### 3.2 Cấu trúc layout

```
┌──────┬────────────────────────────────────┬──────────┐
│      │                                    │          │
│ Side │         STAGE (Player area)        │  Queue   │
│ bar  │   ┌──────────────────────────┐     │  panel   │
│ 64px │   │  Album Art (lớn, glow,   │     │ (toggle) │
│      │   │  + visualizer quanh)     │     │  296px   │
│ vctrl│   │  + now-playing meta      │     │          │
│      │   └──────────────────────────┘     │          │
│ avtr │                                    │          │
│      │   [voice pills floating góc]       │          │
├──────┴────────────────────────────────────┴──────────┤
│         PLAYER BAR (full-width, fixed bottom)        │
│  [thumb] title/artist  ◀ ▶ ▶▶  ━━●━━  🔊  [Q][💬]   │
└──────────────────────────────────────────────────────┘

Chat = overlay dock trượt từ phải (mặc định thu gọn thành icon badge).
```

### 3.3 Các thành phần layout
1. **PlayerBar (full-width, fixed bottom)** — control luôn hiển thị: thumbnail, track title/artist, transport (play/pause/prev/next), seekbar, volume, nút toggle queue & chat. Thay thế "player là panel" hiện tại.
2. **Stage (khu trung tâm)** — hiển thị artwork lớn + metadata + visualizer động quanh artwork. Khi có screenshare/video → Stage chuyển hiển thị video/screenshare, artwork thu nhỏ thành mini trong PlayerBar.
3. **VoicePills (floating)** — thay voice-grid cột cố định 280px. Active speakers thành pill nổi ở góc Stage, tự sắp xếp; pill phóng to khi user đang nói (speaking indicator).
4. **Chat (overlay dock)** — mặc định thu gọn thành icon badge (số tin nhắn chưa đọc). Click mở overlay trượt từ phải, glass blur, không đẩy layout. Đóng khi click ra ngoài hoặc nút close.
5. **Queue (panel phải toggle)** — mặc định mở 296px trên desktop, có nút collapse. Trên tablet/mobile thành bottom sheet trượt.
6. **Sidebar (64px)** — giữ nguyên cấu trúc, re-skin Cool Ocean: voice controls (mic/camera/screen) + avatar (ring pulse khi nói) + back-to-lobby.

### 3.4 Trạng thái layout (state-driven)
Stage mode quyết định hiển thị:
- **`music-only`**: Stage = artwork lớn + visualizer. VoicePills ẩn (không ai trong voice / không ai nói).
- **`music + voice`**: Stage = artwork. VoicePills nổi góc.
- **`screenshare`**: Stage = screen share lớn. Artwork → mini trong PlayerBar. VoicePills nổi góc.
- **`video`**: Stage = video grid lớn. Tương tự screenshare.

### 3.5 Responsive
- Desktop (>1120px): đầy đủ Stage + Queue mở + Sidebar + PlayerBar.
- Tablet (768–1120px): Queue thu thành toggle/sheet.
- Mobile (<768px): Stage thu nhỏ, controls tối giản, Chat + Queue thành bottom sheet full-width, PlayerBar compact.

---

## 4. Component Breakdown (10+, tách mảnh tối đa tái sử dụng)

Tách Room thành các component nhỏ, ranh giới rõ, nhiều phần tái dùng được cho Auth/Dashboard sau.

### 4.1 Shell & layout
- **`RoomShellComponent`** (`room.component`) — container chính; quản lý layout state (stageMode, isChatOpen, isQueueOpen). Thay `room.component` cũ.
- **`RoomStageComponent`** — khu trung tâm; switch giữa artwork view và video/screenshare view theo stageMode.
- **`PlayerBarComponent`** — thanh control full-width bottom; composite các sub-component.
- **`RoomSidebarComponent`** — sidebar 64px (tách từ inline hiện tại trong room.component.html).

### 4.2 Player sub-components (tách mảnh)
- **`ArtworkVisualizerComponent`** — artwork lớn + visualizer động quanh (24–32 bars), halo glow pulsing khi play.
- **`NowPlayingInfoComponent`** — title/artist/album metadata; tái dùng trong cả Stage và PlayerBar thumbnail area.
- **`TransportControlsComponent`** — play/pause/prev/next/shuffle/repeat.
- **`SeekbarComponent`** — seek bar với glow trail khi hover, thumb enlarge, time labels.
- **`VolumeControlComponent`** — volume slider + mute toggle.

### 4.3 Voice & panels
- **`VoicePillsComponent`** — container floating pills; render `VoicePill` cho mỗi participant.
- **`VoicePillComponent`** — pill đơn: avatar (ring pulse khi nói), name, mute indicator; tái dùng `VoicePill`/avatar pattern cho sidebar.
- **`RoomChatComponent`** — giữ logic chat WS, render trong overlay dock; thêm badge unread count; composite message list + input.
- **`RoomQueueComponent`** — giữ logic queue, render trong panel phải toggle; composite queue list + add controls.

### 4.4 UI state cục bộ mới (room.state.ts)
Tách UI state Room khỏi `StateService` (app-wide) để ranh giới rõ:
```ts
RoomUIState {
  stageMode: 'music-only' | 'music-voice' | 'screenshare' | 'video'
  isChatOpen: boolean
  isQueueOpen: boolean
  unreadCount: number
}
```
Playback/voice/chat data vẫn dùng `StateService` + ws services hiện có — không rebuild logic.

### 4.5 Cấu trúc file
```
features/room/
  room.component.{ts,html,css}        → RoomShell (rebuild)
  room.state.ts                       (mới — UI state local)
  components/
    stage/         stage.component.{ts,html,css}        (mới)
    player-bar/    player-bar.component.{ts,html,css}   (mới)
    sidebar/       sidebar.component.{ts,html,css}      (mới — tách inline)
    voice-pills/   voice-pills.component.{ts,html,css}  (mới)
                   voice-pill.component.{ts,html,css}   (mới)
    artwork-visualizer/  ...component.{ts,html,css}     (mới)
    now-playing-info/    ...component.{ts,html,css}     (mới)
    transport-controls/  ...component.{ts,html,css}     (mới)
    seekbar/             ...component.{ts,html,css}     (mới)
    volume-control/      ...component.{ts,html,css}     (mới)
    chat/          chat.component.{ts,html,css}         (adapt overlay)
    queue/         queue.component.{ts,html,css}        (adapt toggle)
  room.component.legacy.{ts,html,css}   (giữ fallback — đổi tên component cũ)
```

---

## 5. Shared Primitives (dùng lại cho Auth/Dashboard sau)

Tạo trong `shared/components/` để chuẩn hóa toàn app:
- **`OceanButtonComponent`** — button với variant primary (gradient teal→indigo), secondary (glass), ghost.
- **`GlassPanelComponent`** — frosted panel dùng `--bg-glass` + blur.
- **`IconButtonComponent`** — icon-only button (dùng cho player controls, sidebar, panel toggles).

Các primitive này được dùng trong Room ngay Phase 1 và tái sử dụng cho Auth/Dashboard ở Phase sau.

---

## 6. Animations & Micro-interactions

### 6.1 Player & artwork
- **Halo glow pulsing**: artwork có glow pulsing nhẹ theo chu kỳ cố định khi `isPlaying` (CSS keyframes, không cần audio analysis).
- **Visualizer**: 24–32 bars quanh artwork (vòng hoặc dải), animate heights. Vì không có raw audio data dễ dàng → **pseudo-visualizer**: heights random với easing, re-seed khi track change / khi `isPlaying` toggle. (Nâng cấp lên real `AnalyserNode` Web Audio là mục tiêu tương lai, ngoài scope Phase 1.)
- **Track change**: artwork crossfade + scale transition (280ms).
- **Seekbar**: glow trail khi hover, thumb enlarge on hover/drag.

### 6.2 Layout transitions
- Chat/Queue open/close: slide + fade (280ms, cubic-bezier expo).
- VoicePill xuất hiện: scale-in + fade; pill phóng to nhẹ khi user đang nói (speaking indicator).
- Stage mode switch (music → screenshare): crossfade giữa artwork view và video view.

### 6.3 Micro-interactions
- Buttons: scale 0.96 on active, gradient glow on hover.
- Cards (queue item, chat message): subtle lift + border glow on hover.
- Toasts: slide-down + glass blur.
- Avatar: ring pulse teal khi user đang nói.
- Loading: shimmer skeleton với ocean gradient (thay spinner).

### 6.4 Tech & a11y
- CSS transitions/keyframes cho phần lớn + `@angular/animations` cho state-driven (panel, stage mode).
- **Respect `prefers-reduced-motion`**: tắt visualizer/halo/pulse, giữ transition thiết yếu (fade ngắn) khi user bật reduced-motion.

---

## 7. Cleanup & Standardization

Thực hiện song song trong quá trình rebuild:
1. **Tokens single source of truth** — xóa hex hardcoded rải rác trong component CSS, thay bằng token.
2. **Typography scale** chuẩn trong tokens (display/lg/md/sm/xs).
3. **Loại bỏ legacy alias** `--accent-alpha` sau khi migrate hết.
4. **Accessibility**:
   - `:focus-visible` rings (teal) trên mọi interactive.
   - aria-label cho icon-only controls (player, sidebar, toggles).
   - Keyboard nav: player controls (Space play/pause, arrows seek/volume), queue (up/down select), chat (Enter send, Esc close).
   - Color contrast AA trên dark navy.
5. **Component mảnh hóa** (10+) giúp Auth/Dashboard tái sử dụng primitives sau.

---

## 8. Ranh giới & Phụ thuộc

- **Giữ nguyên (không rebuild)**: `ApiService`, `ChatWsService`, `PlaybackWsService`, `VoiceService`, `NotificationStreamService`, `StateService` (app-wide), auth.guard, jwt.interceptor, app-shell.
- **Rebuild**: chỉ `features/room/` (component + UI state local).
- **Thêm mới**: design tokens mới, shared primitives, room sub-components.
- **Fallback**: component Room cũ giữ (đổi tên `room.component.legacy`) đến khi Room mới đạt feature-parity & được verify manual, sau đó xóa.

### Tính năng Room phải đạt parity (verify trước khi xóa legacy)
- Voice: join/leave channel, mute/unmute, camera on/off, screen share toggle.
- Playback: play/pause/sync, seek, next/prev, queue add/remove, shuffle/repeat (nếu có).
- Chat: send/receive message realtime, unread tracking.
- State-driven layout: music-only / +voice / screenshare / video.

---

## 9. Phạm vi loại trừ (Out of scope — Phase 1)
- Auth & Dashboard redesign (Phase sau, tái dùng design system & primitives).
- Real audio `AnalyserNode` visualizer (pseudo-visualizer cho Phase 1).
- `web-client-vanilla` (không động vào).
- Backend / API changes.
- Resizable panel (drag-to-resize) — Queue/Chat là toggle cố định.
- Stage fullscreen mode (YAGNI — bỏ nếu chưa cần).

---

## 10. Tiêu chí hoàn thành (Definition of Done) — Phase 1
- [ ] Design tokens Cool Ocean mới apply, không còn hex violet hardcoded trong room.
- [ ] RoomShell rebuild với layout player-centric đầy đủ.
- [ ] 10+ sub-component tạo + dùng được shared primitives.
- [ ] Layout state-driven hoạt động (4 mode Stage).
- [ ] Chat overlay dock + Queue toggle + VoicePills floating hoạt động.
- [ ] Visualizer + halo + micro-interactions, respect `prefers-reduced-motion`.
- [ ] A11y: focus-visible, aria-label, keyboard nav, contrast AA.
- [ ] Feature parity với legacy Room (voice + playback + chat + layout).
- [ ] Responsive desktop/tablet/mobile.
- [ ] Legacy Room component xóa sau khi verify parity.
- [ ] Build pass (`ng build`), không TS error.
