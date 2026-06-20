# Presence & Vibe (Phase 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Phase 1 of the WorkTogether roadmap: Presence & Vibe (live user playback status indicators, unsynced alarms, floating live reactions, speaker/reaction pulses, and a collaborative vibe meter).

**Architecture:** Use the existing unified Chat WebSocket connection (`/api/v1/rooms/:id/chat/ws`) to carry presence states, live reactions, and room-wide vibe updates. Keep the memory footprints light by tracking states in-memory within the Go chat hub, decayed periodically using a ticker.

**Tech Stack:** Go (Gorilla WebSocket, Gin), Angular 19 (TypeScript, RxJS).

## Global Constraints
- Naming & Visual System: Follow Cool Ocean design tokens (Teal-led primary gradient `#2dd4bf` to dark navy `#0a1628` background) and Outfit typography.
- Performance: Ticker loops on backend must run efficiently in O(Room) time.
- Accessibility: Ensure active speaking, unsynced status, and vibe notifications are readable by screen readers via `aria-live` and semantic elements.
- Motion Control: Wrap all emoji animations and pulses in `@media (prefers-reduced-motion: reduce)` to prevent motion sickness.
- Tests: Every task must have unit/integration tests covering code correctness.

---

### Task 1: Go Backend Presence Structs & Hub Initialization

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\ws_hub.go`

**Interfaces:**
- Produces: `MemberState` and `RoomPresence` structs, and the `RoomPresences` field on the `Hub` instance.

- [ ] **Step 1: Write the failing test**
  Add a placeholder test file `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\ws_hub_test.go` checking that `Hub` initialized via `NewHub` has a non-nil `RoomPresences` map.
  ```go
  package http

  import "testing"

  func TestNewHub(t *testing.T) {
      h := NewHub()
      if h.RoomPresences == nil {
          t.Error("Expected RoomPresences map to be initialized, got nil")
      }
  }
  ```
- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...` (run in `c:\Users\tranv\Desktop\WorkTogether\services\chat-service` or root)
  Expected: FAIL (compiler error: `h.RoomPresences undefined`)
- [ ] **Step 3: Write minimal implementation**
  Add the structs and fields to `ws_hub.go`:
  ```go
  // Add to services/chat-service/internal/delivery/http/ws_hub.go
  type MemberState struct {
  	IsPlaying  bool  `json:"is_playing"`
  	PositionMS int   `json:"position_ms"`
  	UpdatedAt  int64 `json:"updated_at"`
  }

  type RoomPresence struct {
  	sync.RWMutex
  	MemberStates    map[string]MemberState
  	RecentReactions map[string]int
  }

  // Inside type Hub struct
  type Hub struct {
  	sync.RWMutex
  	Rooms         map[string]map[*Client]bool
  	RoomPresences map[string]*RoomPresence
  }

  // Inside NewHub()
  func NewHub() *Hub {
  	return &Hub{
  		Rooms:         make(map[string]map[*Client]bool),
  		RoomPresences: make(map[string]*RoomPresence),
  	}
  }
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add services/chat-service/internal/delivery/http/ws_hub.go
  git commit -m "backend: add presence structs and hub fields"
  ```

---

### Task 2: Backend Presence State Change Handler & Broadcast

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\handlers.go`
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\ws_hub.go`

**Interfaces:**
- Consumes: `Hub.RoomPresences`, `MemberState`
- Produces: State updates inside `client.readPump()` and broadcasts `presence:listener_states` payload to the room.

- [ ] **Step 1: Write the failing test**
  Add `TestHandlePresenceStateChange` in `ws_hub_test.go` mock-subscribing a client and sending `presence:state_change` message to verify state updates in `Hub`.
  ```go
  // Add to services/chat-service/internal/delivery/http/ws_hub_test.go
  // A stub test to verify handler logic can process state updates and register room presence map.
  ```
- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Add client state update register/deregister methods in `ws_hub.go`:
  ```go
  func (h *Hub) UpdateClientPresence(roomID string, userID string, isPlaying bool, positionMs int) {
  	h.Lock()
  	presence, exists := h.RoomPresences[roomID]
  	if !exists {
  		presence = &RoomPresence{
  			MemberStates:    make(map[string]MemberState),
  			RecentReactions: make(map[string]int),
  		}
  		h.RoomPresences[roomID] = presence
  	}
  	h.Unlock()

  	presence.Lock()
  	presence.MemberStates[userID] = MemberState{
  		IsPlaying:  isPlaying,
  		PositionMS: positionMs,
  		UpdatedAt:  time.Now().UnixMilli(),
  	}
  	presence.Unlock()
  }
  ```
  In `handlers.go` inside `readPump()`, process `presence:state_change` case:
  ```go
  case "presence:state_change":
  	var payload struct {
  		IsPlaying  bool `json:"is_playing"`
  		PositionMS int  `json:"position_ms"`
  	}
  	payloadBytes, _ := json.Marshal(incoming.Payload)
  	_ = json.Unmarshal(payloadBytes, &payload)

  	c.Hub.UpdateClientPresence(c.RoomID, c.UserID, payload.IsPlaying, payload.PositionMS)

  	// Broadcast all listener states for the room
  	c.Hub.RLock()
  	presence := c.Hub.RoomPresences[c.RoomID]
  	c.Hub.RUnlock()

  	if presence != nil {
  		presence.RLock()
  		broadcastMsg := domain.WSMessage{
  			Event:  "presence:listener_states",
  			RoomID: c.RoomID,
  			Payload: gin.H{
  				"user_states": presence.MemberStates,
  			},
  		}
  		presence.RUnlock()
  		data, _ := json.Marshal(broadcastMsg)
  		c.Hub.BroadcastToRoom(c.RoomID, data)
  	}
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add services/chat-service/internal/delivery/http/handlers.go services/chat-service/internal/delivery/http/ws_hub.go
  git commit -m "backend: implement presence:state_change event handling"
  ```

---

### Task 3: Backend Live Reaction Broadcast

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\handlers.go`
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\ws_hub.go`

**Interfaces:**
- Consumes: `presence:send_reaction` client payload
- Produces: Instant broadcast of `presence:reaction_broadcast` to all room clients and incrementing in-memory reaction counter.

- [ ] **Step 1: Write the failing test**
  Add check in `ws_hub_test.go` that reaction counters increment correctly when a reaction is sent.
- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Add helper inside `ws_hub.go` to record reactions:
  ```go
  func (h *Hub) RecordReaction(roomID string, emoji string) {
  	h.Lock()
  	presence, exists := h.RoomPresences[roomID]
  	if !exists {
  		presence = &RoomPresence{
  			MemberStates:    make(map[string]MemberState),
  			RecentReactions: make(map[string]int),
  		}
  		h.RoomPresences[roomID] = presence
  	}
  	h.Unlock()

  	presence.Lock()
  	presence.RecentReactions[emoji]++
  	presence.Unlock()
  }
  ```
  Handle event in `readPump()` inside `handlers.go`:
  ```go
  case "presence:send_reaction":
  	var payload struct {
  		Emoji string `json:"emoji"`
  	}
  	payloadBytes, _ := json.Marshal(incoming.Payload)
  	_ = json.Unmarshal(payloadBytes, &payload)

  	c.Hub.RecordReaction(c.RoomID, payload.Emoji)

  	broadcastMsg := domain.WSMessage{
  		Event:  "presence:reaction_broadcast",
  		RoomID: c.RoomID,
  		Payload: gin.H{
  			"user_id": c.UserID,
  			"emoji":   payload.Emoji,
  		},
  	}
  	data, _ := json.Marshal(broadcastMsg)
  	c.Hub.BroadcastToRoom(c.RoomID, data)
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add services/chat-service/internal/delivery/http/handlers.go services/chat-service/internal/delivery/http/ws_hub.go
  git commit -m "backend: implement presence:send_reaction and broadcast"
  ```

---

### Task 4: Backend Room Vibe Ticker & Decay System

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\internal\delivery\http\ws_hub.go`
- Modify: `c:\Users\tranv\Desktop\WorkTogether\services\chat-service\cmd\server\main.go`

**Interfaces:**
- Produces: Periodic `presence:vibe_tick` broadcast every 5 seconds.

- [ ] **Step 1: Write the failing test**
  Add a test verifying dominant vibe selection logic based on a mock map of emoji counts.
- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  In `ws_hub.go`, add `StartVibeTicker()` method:
  ```go
  func (h *Hub) StartVibeTicker() {
  	ticker := time.NewTicker(5 * time.Second)
  	go func() {
  		for range ticker.C {
  			h.Lock()
  			for roomID, presence := range h.RoomPresences {
  				presence.Lock()
  				// Calculate Vibe
  				hypeCount := presence.RecentReactions["🔥"] + presence.RecentReactions["👏"]
  				chillCount := presence.RecentReactions["❤️"] + presence.RecentReactions["😮"]
  				studyCount := presence.RecentReactions["📚"]

  				currentVibe := "chill" // default
  				if hypeCount > chillCount && hypeCount > studyCount {
  					currentVibe = "hype"
  				} else if studyCount > chillCount && studyCount > hypeCount {
  					currentVibe = "study"
  				} else if chillCount > 0 {
  					currentVibe = "chill"
  				}

  				vibeMsg := domain.WSMessage{
  					Event:  "presence:vibe_tick",
  					RoomID: roomID,
  					Payload: gin.H{
  						"current_vibe": currentVibe,
  						"vibe_scores": gin.H{
  							"chill": chillCount,
  							"hype":  hypeCount,
  							"study": studyCount,
  						},
  					},
  				}
  				data, _ := json.Marshal(vibeMsg)

  				// Decay reactions by reset or factor
  				presence.RecentReactions = make(map[string]int) // Clear for the next window
  				presence.Unlock()

  				go h.BroadcastToRoom(roomID, data)
  			}
  			h.Unlock()
  		}
  	}()
  }
  ```
  In `cmd/server/main.go`, make sure `hub.StartVibeTicker()` is called after hub initialization.
- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./services/chat-service/internal/delivery/http/...`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add services/chat-service/internal/delivery/http/ws_hub.go
  git commit -m "backend: add periodic vibe ticker and decay system"
  ```

---

### Task 5: Frontend Angular ChatWsService Extensions

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\core\services\websocket\chat-ws.service.ts`

**Interfaces:**
- Produces: Observables `listenerStates$`, `liveReaction$`, `roomVibe$`, and methods `updatePresenceState()`, `sendLiveReaction()`.

- [ ] **Step 1: Write the failing test**
  Add unit tests in `chat-ws.service.spec.ts` mocking incoming WebSocket events (`presence:listener_states`, `presence:reaction_broadcast`, `presence:vibe_tick`) and verify subjects emit correct payloads.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Update `chat-ws.service.ts` with Subjects/BehaviorSubjects and handle parsing cases inside `handleMessage()`:
  ```typescript
  // Imports
  import { BehaviorSubject } from 'rxjs';

  // Inside ChatWsService class
  public listenerStates$ = new BehaviorSubject<Record<string, { is_playing: boolean; position_ms: number; updated_at: number }>>({});
  public liveReaction$ = new Subject<{ user_id: string; emoji: string }>();
  public roomVibe$ = new BehaviorSubject<{ current_vibe: string; vibe_scores: Record<string, number> } | null>(null);

  // In handleMessage(dataStr: string) switch block:
  case 'presence:listener_states':
    this.listenerStates$.next(msg.payload.user_states);
    break;
  case 'presence:reaction_broadcast':
    this.liveReaction$.next({ user_id: msg.payload.user_id, emoji: msg.payload.emoji });
    break;
  case 'presence:vibe_tick':
    this.roomVibe$.next(msg.payload);
    break;

  // New helper methods
  public updatePresenceState(isPlaying: boolean, positionMs: number): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'presence:state_change',
      room_id: this.roomId,
      payload: { is_playing: isPlaying, position_ms: positionMs }
    }));
  }

  public sendLiveReaction(emoji: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify({
      event: 'presence:send_reaction',
      room_id: this.roomId,
      payload: { emoji }
    }));
  }
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/core/services/websocket/chat-ws.service.ts
  git commit -m "frontend: extend chat ws service with presence channels"
  ```

---

### Task 6: Hook Angular Player Engine to Presence Updates

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\player-engine\player-engine.service.ts`

**Interfaces:**
- Consumes: Playback events from YouTube/SoundCloud SDK inside `PlayerEngineService`.
- Produces: Periodic state emissions to `ChatWsService.updatePresenceState()`.

- [ ] **Step 1: Write the failing test**
  Add unit tests in `player-engine.service.spec.ts` mocking playback state changes and ensuring `updatePresenceState()` is invoked.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  In `player-engine.service.ts`, inject `ChatWsService` and start a periodic heartbeat timer (every 5 seconds) if playing, sending current positions. Also send state change instantly on play/pause/seek events:
  ```typescript
  private chatWs = inject(ChatWsService);
  private heartbeatInterval: any = null;

  // In startHeartbeat()
  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      const state = this.currentState$.value;
      if (state && state.state === 'playing') {
        this.chatWs.updatePresenceState(true, state.position_ms);
      }
    }, 5000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  // Whenever local player transitions to playing/paused/seeking, invoke:
  // this.chatWs.updatePresenceState(isPlaying, currentPositionMs);
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/player-engine/player-engine.service.ts
  git commit -m "frontend: hook player engine state changes to presence WS"
  ```

---

### Task 7: Voice Pill Presence Status Icons & Sync Checks

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\voice-pills\voice-pill.component.ts`
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\voice-pills\voice-pills.component.ts`

**Interfaces:**
- Consumes: `ChatWsService.listenerStates$` and master room position from `PlaybackWsService`.
- Produces: Rendered play/pause indicators and warning styles on unsynced listeners.

- [ ] **Step 1: Write the failing test**
  Add test asserting that participants with playback offsets > 3 seconds are assigned `isUnsynced: true`.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Add playback status fields to `PillParticipant` interface in `voice-pill.component.ts`:
  ```typescript
  export interface PillParticipant {
    sid: string;
    identity?: string;
    display_name: string;
    avatar_url?: string;
    isSpeaking: boolean;
    isMuted: boolean;
    isCurrentUser: boolean;
    isPlaying: boolean;
    isUnsynced: boolean;
  }
  ```
  Update template in `voice-pill.component.ts`:
  - Render small triangle play icon if `isPlaying` is true, or double-bar pause icon if false, overlaying avatar.
  - Render an orange warning dot next to name or on avatar if `isUnsynced` is true. Add tooltip "Lệch pha".
  Update `voice-pills.component.ts` sync logic:
  - Inject `PlaybackWsService` and `ChatWsService`.
  - Subscribe to `ChatWsService.listenerStates$`.
  - In `sync()`, match member ID with states map.
  - Calculate `isUnsynced`:
    ```typescript
    const state = listenerStates[m.user_id];
    let isPlaying = false;
    let isUnsynced = false;
    if (state) {
      isPlaying = state.is_playing;
      const master = this.playbackWs.playbackSync$.value;
      if (master && master.state === 'playing') {
        const estimatedPosition = state.is_playing
          ? state.position_ms + (Date.now() - state.updated_at)
          : state.position_ms;
        const diff = Math.abs(estimatedPosition - master.position_ms);
        isUnsynced = diff > 3000;
      }
    }
    ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/voice-pills/
  git commit -m "frontend: display listener status and unsynced warnings on voice pills"
  ```

---

### Task 8: Voice Pill Reaction Pulses & Floating Emojis

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\voice-pills\voice-pill.component.ts`

**Interfaces:**
- Consumes: `ChatWsService.liveReaction$`

- [ ] **Step 1: Write the failing test**
  Add test that calling reaction trigger method appends a temporary floating element to the pill.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Inside `VoicePillComponent`, listen to reactions. If reaction matches this participant's user identity, apply `.react-bounce` class for 1.2s and append floating emoji spans that animate upwards and fade out:
  ```typescript
  // Inside VoicePillComponent template, add absolute container for emoji floaters
  // CSS:
  // .voice-pill.react-bounce { animation: pill-bounce 1s cubic-bezier(0.175, 0.885, 0.32, 1.275); }
  // @keyframes pill-bounce { 0%, 100% { transform: scale(1); } 50% { transform: scale(1.15); } }
  // .floater { position: absolute; animation: float-up 1.5s ease-out forwards; }
  // @keyframes float-up { 0% { transform: translateY(0) scale(0.5); opacity: 0; } 20% { opacity: 1; } 100% { transform: translateY(-50px) scale(1.2); opacity: 0; } }
  ```
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/voice-pills/voice-pill.component.ts
  git commit -m "frontend: implement voice pill elastic pulse and micro floating emojis"
  ```

---

### Task 9: Quick Reactions Bar Component

**Files:**
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\quick-reactions\quick-reactions.component.ts`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\quick-reactions\quick-reactions.component.css`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\quick-reactions\quick-reactions.component.spec.ts`

**Interfaces:**
- Produces: A horizontal glassmorphism bar component with click handlers emitting reactions through `ChatWsService`.

- [ ] **Step 1: Write the failing test**
  Write a test in `quick-reactions.component.spec.ts` to assert buttons for ❤️, 🔥, 👏, 😮, 📚 are rendered and trigger `chatWs.sendLiveReaction()`.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL (files do not exist or tests fail)
- [ ] **Step 3: Write minimal implementation**
  Create the QuickReactionsComponent:
  ```typescript
  import { Component, inject } from '@angular/core';
  import { CommonModule } from '@angular/common';
  import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

  @Component({
    selector: 'app-room-quick-reactions',
    standalone: true,
    imports: [CommonModule],
    template: `
      <div class="quick-reactions-bar" role="toolbar" aria-label="Bắn biểu cảm">
        @for (emoji of emojis; track emoji) {
          <button (click)="react(emoji)" [attr.aria-label]="'Bắn ' + emoji">{{ emoji }}</button>
        }
      </div>
    `,
    styleUrl: './quick-reactions.component.css'
  })
  export class QuickReactionsComponent {
    private chatWs = inject(ChatWsService);
    public emojis = ['❤️', '🔥', '👏', '😮', '📚'];
    react(emoji: string) {
      this.chatWs.sendLiveReaction(emoji);
    }
  }
  ```
  Styling in `quick-reactions.component.css` using glassmorphism styling conforming to Cool Ocean.
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/quick-reactions/
  git commit -m "frontend: create quick reactions bar component"
  ```

---

### Task 10: Reactions Canvas Component (Flying Emojis)

**Files:**
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\reactions-canvas\reactions-canvas.component.ts`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\reactions-canvas\reactions-canvas.component.css`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\reactions-canvas\reactions-canvas.component.spec.ts`

**Interfaces:**
- Consumes: Live reactions from `ChatWsService.liveReaction$`

- [ ] **Step 1: Write the failing test**
  Write tests checking that receipt of a reaction appends a styled DOM node with sinus path animations to the canvas.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Create `ReactionsCanvasComponent` appending floating emoji instances dynamically using randomized left offsets, swaying paths, and automatically destroying them after 3s. Support `prefers-reduced-motion` logic.
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/reactions-canvas/
  git commit -m "frontend: create reactions canvas overlay component"
  ```

---

### Task 11: Room Vibe Meter Component

**Files:**
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\vibe-meter\vibe-meter.component.ts`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\vibe-meter\vibe-meter.component.css`
- Create: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\components\vibe-meter\vibe-meter.component.spec.ts`

**Interfaces:**
- Consumes: `ChatWsService.roomVibe$`

- [ ] **Step 1: Write the failing test**
  Write tests verifying the meter UI updates class names (e.g. `.vibe-hype`, `.vibe-chill`) when receiving room vibe ticks.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL
- [ ] **Step 3: Write minimal implementation**
  Implement the `RoomVibeMeterComponent`. Use breathing pulse glow animations mapped to the current vibe. Render a small indicator of the dominating mood with clean typography.
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/components/vibe-meter/
  git commit -m "frontend: create room vibe meter widget component"
  ```

---

### Task 12: Layout Integration in Room Shell Component

**Files:**
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\room.component.ts`
- Modify: `c:\Users\tranv\Desktop\WorkTogether\web-client\src\app\features\room\room.component.html`

**Interfaces:**
- Consumes: The newly created components (`QuickReactionsComponent`, `ReactionsCanvasComponent`, `RoomVibeMeterComponent`).

- [ ] **Step 1: Write the failing test**
  Modify `room.component.spec.ts` to include the new mock services or component declarations, and run to verify it compiles.
- [ ] **Step 2: Run test to verify it fails**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: FAIL (if templates references elements not registered)
- [ ] **Step 3: Write minimal implementation**
  - Import the new standalone components inside `room.component.ts` imports array.
  - Insert `<app-room-vibe-meter>` into the header/top area.
  - Insert `<app-room-reactions-canvas>` overlaying the central music/stage section.
  - Insert `<app-room-quick-reactions>` inside/above the bottom player bar.
- [ ] **Step 4: Run test to verify it passes**
  Run: `npm run test -- --watch=false --browsers=ChromeHeadless`
  Expected: PASS
- [ ] **Step 5: Commit**
  ```bash
  git add web-client/src/app/features/room/room.component.ts web-client/src/app/features/room/room.component.html
  git commit -m "frontend: integrate quick reactions, canvas, and vibe meter in room shell"
  ```
