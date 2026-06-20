# Room Redesign (Cool Ocean, Player-Centric) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the Angular 19 Room view with a Cool Ocean design system and a Spotify-like player-centric layout, preserving all existing voice/playback/chat logic.

**Architecture:** Approach B — rebuild only `features/room/` components + UI state; keep all core services (`ApiService`, `ChatWsService`, `PlaybackWsService`, `VoiceService`, `StateService`) untouched and port their logic. Build a new Cool Ocean design-token layer over the existing one, add shared primitives, and split Room into 12 focused standalone components. The legacy `RoomComponent` is renamed and kept as fallback until parity is verified, then removed.

**Tech Stack:** Angular 19 (standalone components, `@if`/`@for` control flow, signals optional but prefer current BehaviorSubject pattern), TypeScript 5.7, CSS custom properties (design tokens), `@angular/animations`, livekit-client 2.16 (unchanged), YouTube/SoundCloud/HTML5 player APIs (unchanged).

**Spec:** `docs/superpowers/specs/2026-06-20-room-redesign-cool-ocean-design.md`

---

## File Structure

This plan touches/creates the following. Each file has one responsibility.

**Design tokens & global (modify):**
- `web-client/src/design-tokens.css` — rewrite with Cool Ocean palette (keep legacy aliases temporarily).

**Shared primitives (new, reusable for Auth/Dashboard later):**
- `web-client/src/app/shared/components/ocean-button/ocean-button.component.ts` — button with variants.
- `web-client/src/app/shared/components/glass-panel/glass-panel.component.ts` — frosted container.
- `web-client/src/app/shared/components/icon-button/icon-button.component.ts` — icon-only button.

**Room UI state (new):**
- `web-client/src/app/features/room/room-ui-state.service.ts` — local UI state (panels, stage mode, unread).

**Room shell + layout (rebuild):**
- `web-client/src/app/features/room/room.component.{ts,html,css}` — `RoomShell` (rebuild; old renamed to `.legacy`).
- `web-client/src/app/features/room/components/sidebar/sidebar.component.{ts,html,css}` — 64px sidebar (extracted from inline).
- `web-client/src/app/features/room/components/stage/stage.component.{ts,html,css}` — center stage.
- `web-client/src/app/features/room/components/player-bar/player-bar.component.{ts,html,css}` — full-width bottom control bar.

**Player sub-components (new, extracted):**
- `web-client/src/app/features/room/components/artwork-visualizer/artwork-visualizer.component.{ts,html,css}`
- `web-client/src/app/features/room/components/now-playing-info/now-playing-info.component.{ts,html,css}`
- `web-client/src/app/features/room/components/transport-controls/transport-controls.component.{ts,html,css}`
- `web-client/src/app/features/room/components/seekbar/seekbar.component.{ts,html,css}`
- `web-client/src/app/features/room/components/volume-control/volume-control.component.{ts,html,css}`

**Voice & panels (new / adapted):**
- `web-client/src/app/features/room/components/voice-pills/voice-pills.component.{ts,html,css}`
- `web-client/src/app/features/room/components/voice-pills/voice-pill.component.{ts,html,css}`
- `web-client/src/app/features/room/components/chat/chat.component.{ts,html,css}` — adapted to overlay dock + unread badge.
- `web-client/src/app/features/room/components/queue/queue.component.{ts,html,css}` — adapted to toggle panel.

**Player engine service (new):** The existing `player.component.ts` mixes player-sync engine logic (YouTube/SoundCloud/HTML5 sync) with UI. Extract the engine into a pure service so the new UI components stay focused:
- `web-client/src/app/features/room/components/player-engine/player-engine.service.ts` — playback sync engine (port of `PlayerComponent` engine methods).

**Tests:**
- `web-client/src/app/shared/components/ocean-button/ocean-button.component.spec.ts`
- `web-client/src/app/features/room/room-ui-state.service.spec.ts`
- `web-client/src/app/features/room/components/seekbar/seekbar.component.spec.ts`
- `web-client/src/app/features/room/components/player-engine/player-engine.service.spec.ts`
- (existing) `web-client/src/app/features/room/room.component.spec.ts`

---

## Conventions used throughout this plan

- **Standalone components** with `imports: [CommonModule, ...]`, inline or external templates per file size (small → inline `template`, larger → `templateUrl`). Match existing code.
- **Signals vs BehaviorSubject:** codebase uses `BehaviorSubject` + `async` pipe extensively. Follow that; do not introduce signals mid-redesign.
- **Selector prefix:** `app-`. Feature sub-components keep selector scoped to feature (e.g. `app-room-stage`).
- **CSS:** component-scoped (`styleUrl`); rely on design tokens from `design-tokens.css`. Never hardcode hex in component CSS.
- **A11y:** every icon-only `<button>` gets `[title]` + `aria-label`; interactive controls have `:focus-visible` styling.
- **Commit cadence:** commit after each task's green step. Prefix: `feat(room)`, `feat(design)`, `refactor(room)`, `style(room)`, `test(room)`, `chore(room)`.
- **Run tests:** `npx ng test --watch=false --browsers=ChromeHeadless` (from `web-client/`). Build check: `npx ng build`.

---

## Task 1: Establish Cool Ocean design tokens

**Files:**
- Modify: `web-client/src/design-tokens.css`

- [ ] **Step 1: Rewrite design-tokens.css with Cool Ocean palette**

Replace the entire `:root` block. Keep `--accent-alpha` as a temporary alias pointing to teal so unrebuilt components don't break. Add the typography scale and panel transition var.

```css
/* ============================================================
   WorkTogether — Design Tokens
   Cool Ocean system. Dark navy base, teal-led gradient accent.
   ============================================================ */

:root {
  /* ── Backgrounds (deep ocean navy) ──────────────────────── */
  --bg-base:       #0a1628;
  --bg-primary:    #0a1628;   /* alias kept for components using old name */
  --bg-surface:    #0f2240;
  --bg-secondary:  #0f2240;   /* alias kept for components using old name */
  --bg-elevated:   #163355;
  --bg-hover:      #163355;   /* alias kept */
  --bg-glass:      rgba(15, 34, 64, 0.6);

  /* ── Borders ─────────────────────────────────────────────── */
  --border-color:  #1b3a5c;
  --border-focus:  #2dd4bf;

  /* ── Accent (ocean gradient, teal-led) ───────────────────── */
  --ocean-teal:    #2dd4bf;
  --ocean-sky:     #38bdf8;
  --ocean-indigo:  #6366f1;
  --accent-primary:   #2dd4bf;
  --accent-hover:     #34e0cc;
  --accent-dim:       rgba(45, 212, 191, 0.12);
  --accent-glow:      0 0 20px rgba(45, 212, 191, 0.18);
  --accent-glow-color: rgba(45, 212, 191, 0.18);
  --accent-gradient:  linear-gradient(135deg, #2dd4bf 0%, #38bdf8 50%, #6366f1 100%);

  /* Legacy alias — temporary, removed after full migration (Task 16) */
  --accent-alpha:   rgba(45, 212, 191, 0.12);

  /* ── Text ────────────────────────────────────────────────── */
  --text-primary:   #e6f1ff;
  --text-secondary: #8aa4c8;
  --text-muted:     #4a6080;

  /* ── Semantic ────────────────────────────────────────────── */
  --success: #2dd4bf;
  --warning: #fbbf24;
  --danger:  #f87171;

  /* ── Typography ──────────────────────────────────────────── */
  --font-sans: 'Outfit', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'JetBrains Mono', monospace;

  --text-display-size: 2rem;   /* 32px */
  --text-lg-size:      1.25rem;/* 20px */
  --text-md-size:      1rem;   /* 16px */
  --text-sm-size:      0.875rem;/* 14px */
  --text-xs-size:      0.75rem;/* 12px */
  --leading-tight: 1.2;
  --leading-normal: 1.5;

  /* ── Radius (fresh, larger) ──────────────────────────────── */
  --radius-sm:   6px;
  --radius-md:   10px;
  --radius-lg:   14px;
  --radius-xl:   22px;
  --radius-full: 9999px;

  /* ── Motion ──────────────────────────────────────────────── */
  --ease-out-expo: cubic-bezier(0.16, 1, 0.3, 1);
  --transition-fast:   all 0.15s var(--ease-out-expo);
  --transition-normal: all 0.25s var(--ease-out-expo);
  --transition-slow:   all 0.4s  var(--ease-out-expo);
  --transition-panel:  all 0.28s var(--ease-out-expo);

  /* ── Shadows (bg-hue tinted) ─────────────────────────────── */
  --shadow-sm: 0 1px 3px rgba(4, 10, 22, 0.5);
  --shadow-md: 0 4px 12px rgba(4, 10, 22, 0.6);
  --shadow-lg: 0 12px 32px rgba(4, 10, 22, 0.7), 0 0 0 1px rgba(45, 212, 191, 0.04);
  --shadow-accent: 0 0 20px rgba(45, 212, 191, 0.2), 0 4px 12px rgba(4, 10, 22, 0.6);

  /* ── Bezels ──────────────────────────────────────────────── */
  --inner-bezel: inset 0 1px 0 0 rgba(255, 255, 255, 0.05);
  --outer-bezel: 0 0 0 1px var(--border-color);

  /* ── Z-index scale ───────────────────────────────────────── */
  --z-base:    1;
  --z-overlay: 100;
  --z-modal:   200;
  --z-toast:   300;
  --z-player:  10;
}
```

- [ ] **Step 2: Verify build still passes**

Run: `npx ng build`
Expected: builds successfully. (Component CSS still references tokens that exist via aliases; nothing removed.)

- [ ] **Step 3: Commit**

```bash
git add web-client/src/design-tokens.css
git commit -m "feat(design): introduce Cool Ocean design tokens"
```

---

## Task 2: Global reduced-motion + a11y baseline

**Files:**
- Modify: `web-client/src/styles.css`

- [ ] **Step 1: Append reduced-motion + focus-visible rules to styles.css**

Add at the end of `web-client/src/styles.css` (after the existing scrollbar block):

```css
/* ── Accessibility: focus-visible rings (Cool Ocean teal) ── */
:focus-visible {
  outline: 2px solid var(--accent-primary);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}

/* ── Reduced motion: disable decorative animation ───────── */
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.001ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.001ms !important;
    scroll-behavior: auto !important;
  }
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/styles.css
git commit -m "feat(design): add focus-visible rings and reduced-motion baseline"
```

---

## Task 3: Shared primitive — IconButtonComponent

Start with the simplest shared primitive; the others (OceanButton, GlassPanel) are added in Tasks 4–5 following the same pattern.

**Files:**
- Create: `web-client/src/app/shared/components/icon-button/icon-button.component.ts`
- Create: `web-client/src/app/shared/components/icon-button/icon-button.component.spec.ts`

- [ ] **Step 1: Write the failing test**

`web-client/src/app/shared/components/icon-button/icon-button.component.spec.ts`:

```typescript
import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { IconButtonComponent } from './icon-button.component';

describe('IconButtonComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('renders an icon-only button with title and aria-label', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Play music';
    cmp.disabled = false;
    fixture.detectChanges();

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    expect(btn.getAttribute('aria-label')).toBe('Play music');
    expect(btn.title).toBe('Play music');
    expect(btn.disabled).toBeFalse();
  });

  it('emits clicked when pressed and not disabled', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Mute';
    cmp.disabled = false;
    fixture.detectChanges();

    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    btn.click();
    expect(fired).toBeTrue();
  });

  it('does not emit when disabled', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Mute';
    cmp.disabled = true;
    fixture.detectChanges();

    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    btn.click();
    expect(fired).toBeFalse();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/icon-button.component.spec.ts"`
Expected: FAIL (component not defined / no `clicked` output).

- [ ] **Step 3: Implement IconButtonComponent**

`web-client/src/app/shared/components/icon-button/icon-button.component.ts`:

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/**
 * Icon-only button. Always sets an accessible label via `label`.
 * Variants: 'default' | 'active' | 'danger'.
 */
@Component({
  selector: 'app-icon-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      type="button"
      class="icon-btn"
      [class.active]="variant === 'active'"
      [class.danger]="variant === 'danger'"
      [disabled]="disabled"
      [attr.aria-label]="label"
      [attr.title]="label"
      (click)="onClick()"
    >
      <ng-content></ng-content>
    </button>
  `,
  styles: [`
    .icon-btn {
      width: 40px; height: 40px;
      display: inline-flex; align-items: center; justify-content: center;
      border: none; border-radius: var(--radius-md);
      background: transparent; color: var(--text-secondary);
      cursor: pointer; transition: var(--transition-fast);
    }
    .icon-btn:hover:not(:disabled) {
      background: var(--bg-hover); color: var(--text-primary); transform: scale(1.05);
    }
    .icon-btn:active:not(:disabled) { transform: scale(0.96); }
    .icon-btn.active {
      background: var(--accent-dim); color: var(--accent-primary);
      box-shadow: inset 0 0 0 1px rgba(45, 212, 191, 0.25);
    }
    .icon-btn.danger { color: var(--danger); }
    .icon-btn:disabled { opacity: 0.45; cursor: not-allowed; }
    ::ng-deep .icon-btn svg { width: 20px; height: 20px; }
  `]
})
export class IconButtonComponent {
  @Input() label = '';
  @Input() disabled = false;
  @Input() variant: 'default' | 'active' | 'danger' = 'default';
  @Output() clicked = new EventEmitter<void>();

  onClick(): void {
    if (!this.disabled) {
      this.clicked.emit();
    }
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/icon-button.component.spec.ts"`
Expected: PASS (3 specs).

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/shared/components/icon-button/
git commit -m "feat(shared): add IconButtonComponent primitive"
```

---

## Task 4: Shared primitive — GlassPanelComponent

**Files:**
- Create: `web-client/src/app/shared/components/glass-panel/glass-panel.component.ts`

- [ ] **Step 1: Implement GlassPanelComponent**

```typescript
import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Frosted-glass container using --bg-glass + backdrop blur. */
@Component({
  selector: 'app-glass-panel',
  standalone: true,
  imports: [CommonModule],
  template: `<div class="glass-panel" [class.elevated]="elevated"><ng-content></ng-content></div>`,
  styles: [`
    .glass-panel {
      background: var(--bg-glass);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid var(--border-color);
      border-radius: var(--radius-lg);
    }
    .glass-panel.elevated { box-shadow: var(--shadow-lg); }
  `]
})
export class GlassPanelComponent {
  @Input() elevated = false;
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/app/shared/components/glass-panel/
git commit -m "feat(shared): add GlassPanelComponent primitive"
```

---

## Task 5: Shared primitive — OceanButtonComponent

**Files:**
- Create: `web-client/src/app/shared/components/ocean-button/ocean-button.component.ts`
- Create: `web-client/src/app/shared/components/ocean-button/ocean-button.component.spec.ts`

- [ ] **Step 1: Write the failing test**

```typescript
import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { OceanButtonComponent } from './ocean-button.component';

describe('OceanButtonComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('applies variant class and is disabled stateful', () => {
    const fixture = TestBed.createComponent(OceanButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.variant = 'primary';
    cmp.disabled = true;
    fixture.detectChanges();

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    expect(btn.classList.contains('primary')).toBeTrue();
    expect(btn.disabled).toBeTrue();
  });

  it('emits clicked when not disabled', () => {
    const fixture = TestBed.createComponent(OceanButtonComponent);
    const cmp = fixture.componentInstance;
    fixture.detectChanges();
    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));
    fixture.debugElement.query(By.css('button')).nativeElement.click();
    expect(fired).toBeTrue();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/ocean-button.component.spec.ts"`
Expected: FAIL.

- [ ] **Step 3: Implement OceanButtonComponent**

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** App button. Variants: 'primary' (gradient), 'secondary' (glass), 'ghost'. */
@Component({
  selector: 'app-ocean-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      type="button"
      class="ocean-btn"
      [class.primary]="variant === 'primary'"
      [class.secondary]="variant === 'secondary'"
      [class.ghost]="variant === 'ghost'"
      [disabled]="disabled"
      (click)="onClick()"
    >
      <ng-content></ng-content>
    </button>
  `,
  styles: [`
    .ocean-btn {
      font-family: var(--font-sans); font-weight: 600;
      padding: 10px 18px; border: none; border-radius: var(--radius-full);
      cursor: pointer; transition: var(--transition-fast);
      color: var(--text-primary);
    }
    .ocean-btn.primary {
      background: var(--accent-gradient); color: #05121f;
      box-shadow: var(--shadow-accent);
    }
    .ocean-btn.primary:hover:not(:disabled) { transform: translateY(-1px); }
    .ocean-btn.secondary {
      background: var(--bg-glass); backdrop-filter: blur(12px);
      border: 1px solid var(--border-color);
    }
    .ocean-btn.secondary:hover:not(:disabled) { border-color: var(--border-focus); }
    .ocean-btn.ghost { background: transparent; color: var(--text-secondary); }
    .ocean-btn.ghost:hover:not(:disabled) { color: var(--text-primary); background: var(--bg-hover); }
    .ocean-btn:active:not(:disabled) { transform: scale(0.97); }
    .ocean-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  `]
})
export class OceanButtonComponent {
  @Input() variant: 'primary' | 'secondary' | 'ghost' = 'primary';
  @Input() disabled = false;
  @Output() clicked = new EventEmitter<void>();

  onClick(): void {
    if (!this.disabled) this.clicked.emit();
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/ocean-button.component.spec.ts"`
Expected: PASS (2 specs).

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/shared/components/ocean-button/
git commit -m "feat(shared): add OceanButtonComponent primitive"
```

---

## Task 6: RoomUiState service (local UI state)

This isolates Room UI state (panels, stage mode, unread) from the app-wide `StateService`. Voice/playback/chat data continue to come from existing services.

**Files:**
- Create: `web-client/src/app/features/room/room-ui-state.service.ts`
- Create: `web-client/src/app/features/room/room-ui-state.service.spec.ts`

- [ ] **Step 1: Write the failing test**

`web-client/src/app/features/room/room-ui-state.service.spec.ts`:

```typescript
import { TestBed } from '@angular/core/testing';
import { RoomUiStateService } from './room-ui-state.service';

describe('RoomUiStateService', () => {
  let svc: RoomUiStateService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    svc = TestBed.inject(RoomUiStateService);
  });

  it('starts closed with music-only stage and zero unread', () => {
    expect(svc.uiState.isChatOpen).toBeFalse();
    expect(svc.uiState.isQueueOpen).toBeTrue();
    expect(svc.uiState.stageMode).toBe('music-only');
    expect(svc.uiState.unreadCount).toBe(0);
  });

  it('toggles chat and resets unread when opened', () => {
    svc.markUnread(3);
    expect(svc.uiState.unreadCount).toBe(3);

    svc.toggleChat(true);
    expect(svc.uiState.isChatOpen).toBeTrue();
    expect(svc.uiState.unreadCount).toBe(0);
  });

  it('setStageMode ignores unknown mode', () => {
    svc.setStageMode('screenshare');
    expect(svc.uiState.stageMode).toBe('screenshare');
    // @ts-expect-error invalid mode
    svc.setStageMode('nope');
    expect(svc.uiState.stageMode).toBe('screenshare');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/room-ui-state.service.spec.ts"`
Expected: FAIL (service not defined).

- [ ] **Step 3: Implement RoomUiStateService**

```typescript
import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

export type StageMode = 'music-only' | 'music-voice' | 'screenshare' | 'video';

export interface RoomUIState {
  stageMode: StageMode;
  isChatOpen: boolean;
  isQueueOpen: boolean;
  unreadCount: number;
}

const VALID_MODES: StageMode[] = ['music-only', 'music-voice', 'screenshare', 'video'];

/** Local Room UI state, isolated from app-wide StateService. */
@Injectable({ providedIn: 'root' })
export class RoomUiStateService {
  private readonly state$ = new BehaviorSubject<RoomUIState>({
    stageMode: 'music-only',
    isChatOpen: false,
    isQueueOpen: true,
    unreadCount: 0
  });

  public get uiState(): RoomUIState {
    return this.state$.value;
  }

  public get changes$() {
    return this.state$.asObservable();
  }

  public toggleChat(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isChatOpen : open;
    this.patch({ isChatOpen: next, unreadCount: next ? 0 : this.state$.value.unreadCount });
  }

  public toggleQueue(open?: boolean): void {
    const next = open === undefined ? !this.state$.value.isQueueOpen : open;
    this.patch({ isQueueOpen: next });
  }

  public markUnread(n: number): void {
    this.patch({ unreadCount: this.state$.value.unreadCount + n });
  }

  public setStageMode(mode: StageMode): void {
    if (!VALID_MODES.includes(mode)) return;
    this.patch({ stageMode: mode });
  }

  private patch(partial: Partial<RoomUIState>): void {
    this.state$.next({ ...this.state$.value, ...partial });
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/room-ui-state.service.spec.ts"`
Expected: PASS (3 specs).

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/room-ui-state.service.ts web-client/src/app/features/room/room-ui-state.service.spec.ts
git commit -m "feat(room): add RoomUiState service for local UI state"
```

---

## Task 7: Extract PlayerEngine service (pure playback sync)

The existing `PlayerComponent` (~540 lines) mixes the YouTube/SoundCloud/HTML5 sync engine with UI. Extract the engine into a pure service so new UI components stay focused and the engine is unit-testable without a DOM player.

The engine ports these methods verbatim from `PlayerComponent`: `initYoutubePlayer`, `initSoundCloudPlayer`, `initHtml5Audio`, `syncPlayerWithServerState`, `syncYoutubePlayer`, `syncSoundCloudPlayer`, `syncHtml5Audio`, `handlePlaybackSync`, `handleScStateChange`, `handleHtml5StateChange`, `handlePlayerStateChange`, `stopAllPlayers`, plus the player element-id helpers. It exposes observables for `currentState`, `currentTrack`, `displayProgress`, `displayProgressPct`, `volume`.

**Files:**
- Create: `web-client/src/app/features/room/components/player-engine/player-engine.service.ts`
- Create: `web-client/src/app/features/room/components/player-engine/player-engine.service.spec.ts`

- [ ] **Step 1: Write the failing test (behavioral — engine drives progress + commands)**

`player-engine.service.spec.ts`:

```typescript
import { TestBed } from '@angular/core/testing';
import { PlayerEngineService } from './player-engine.service';
import { PlaybackWsService, PlaybackState } from '../../../../core/services/websocket/playback-ws.service';

describe('PlayerEngineService', () => {
  let engine: PlayerEngineService;
  let playbackWs: PlaybackWsService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    engine = TestBed.inject(PlayerEngineService);
    playbackWs = TestBed.inject(PlaybackWsService);
  });

  it('reflects playback sync state and resets on null track', () => {
    const state: PlaybackState = {
      state: 'playing',
      current_track: {
        id: 't1', title: 'Song', artist: 'A', thumbnail_url: '',
        duration_ms: 10000, source_url: 'https://youtu.be/abc', source: 'youtube'
      },
      position_ms: 1000,
      updated_at: Date.now()
    };
    playbackWs.playbackSync$.next(state);
    expect(engine.currentTrack?.id).toBe('t1');
    expect(engine.currentState).toBe('playing');
  });

  it('sends play/pause control command via PlaybackWsService', () => {
    const spy = spyOn(playbackWs, 'sendControlCommand');
    // seed a track so command is not short-circuited
    playbackWs.playbackSync$.next({
      state: 'paused', current_track: { id: 't1', title: 'x', artist: 'y', thumbnail_url: '', duration_ms: 5000, source_url: 'u', source: 'youtube' },
      position_ms: 0, updated_at: Date.now()
    });
    engine.togglePlayPause();
    expect(spy).toHaveBeenCalledWith('play', jasmine.any(Number));
  });

  it('computes seek ms from a scrub fraction', () => {
    playbackWs.playbackSync$.next({
      state: 'paused', current_track: { id: 't1', title: 'x', artist: 'y', thumbnail_url: '', duration_ms: 60000, source_url: 'u', source: 'youtube' },
      position_ms: 0, updated_at: Date.now()
    });
    const seekMs = engine.seekMsFromFraction(0.5);
    expect(seekMs).toBe(30000);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/player-engine.service.spec.ts"`
Expected: FAIL (service not defined).

- [ ] **Step 3: Implement PlayerEngineService (port engine logic)**

Port the engine methods from the existing `player.component.ts`. The service keeps the same YouTube/SoundCloud/HTML5 element ids (`youtube-player-element`, `soundcloud-player-element`, `html5-audio-element`) so the rendered iframe/audio elements in the new PlayerBar/Stage can host them. UI state (`displayProgress`, `displayProgressPct`, `volume`, `currentState`, `currentTrack`) is exposed as observables + getters.

`player-engine.service.ts`:

```typescript
import { Injectable, NgZone, OnDestroy, inject } from '@angular/core';
import { BehaviorSubject, Subscription } from 'rxjs';
import { PlaybackWsService, PlaybackState, Track } from '../../../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../../../core/services/voice.service';

/**
 * Pure playback-sync engine extracted from the legacy PlayerComponent.
 * Hosts the YouTube IFrame / SoundCloud Widget / HTML5 audio players by element id.
 * UI components read observables here and call command methods.
 */
@Injectable({ providedIn: 'root' })
export class PlayerEngineService implements OnDestroy {
  private playbackWs = inject(PlaybackWsService);
  private voiceService = inject(VoiceService);
  private zone = inject(NgZone);

  private ytPlayer: any = null;
  private isReady = false;
  private isSyncingLocal = false;
  private progressTimer: any = null;
  private scWidget: any = null;
  private isScReady = false;
  private audioEl: HTMLAudioElement | null = null;

  private readonly _currentState$ = new BehaviorSubject<'playing' | 'paused' | 'stopped'>('stopped');
  private readonly _currentTrack$ = new BehaviorSubject<Track | null>(null);
  private readonly _progressMs$ = new BehaviorSubject<number>(0);
  private readonly _progressPct$ = new BehaviorSubject<number>(0);
  private readonly _volume$ = new BehaviorSubject<number>(80);
  private readonly _hasScreenshare$ = new BehaviorSubject<boolean>(false);

  private lastServerPosition = 0;
  private lastUpdatedAt = 0;
  private subs: Subscription[] = [];

  constructor() {
    this.subs.push(
      this.playbackWs.playbackSync$.subscribe(state => {
        if (state) this.handlePlaybackSync(state);
      }),
      this.voiceService.participants$.subscribe(list => {
        this._hasScreenshare$.next((list || []).some(p => p.isScreenSharing));
      })
    );
  }

  /* ── Public observables / getters ───────────────────────── */
  public get currentState() { return this._currentState$.value; }
  public get currentTrack() { return this._currentTrack$.value; }
  public get volume() { return this._volume$.value; }
  public get currentState$() { return this._currentState$.asObservable(); }
  public get currentTrack$() { return this._currentTrack$.asObservable(); }
  public get progressMs$() { return this._progressMs$.asObservable(); }
  public get progressPct$() { return this._progressPct$.asObservable(); }
  public get volume$() { return this._volume$.asObservable(); }
  public get hasScreenshare$() { return this._hasScreenshare$.asObservable(); }

  /* ── Bootstrap (called by host component AfterViewInit) ─── */
  public loadApis(): void {
    if (!(window as any)['YT'] || !(window as any)['YT'].Player) {
      const tag = document.createElement('script');
      tag.src = 'https://www.youtube.com/iframe_api';
      document.getElementsByTagName('script')[0].parentNode?.insertBefore(tag, document.getElementsByTagName('script')[0]);
    }
    if (!(window as any)['SC']) {
      const tag = document.createElement('script');
      tag.src = 'https://w.soundcloud.com/player/api.js';
      document.head.appendChild(tag);
    }
  }

  public initPlayers(): void {
    this.initYoutubePlayer();
    this.initSoundCloudPlayer();
    this.initHtml5Audio();
  }

  /* ── Engine internals (ported from PlayerComponent) ─────── */
  private safeCall(method: string, args: any[] = [], defaultValue: any = null): any {
    if (this.ytPlayer && typeof this.ytPlayer[method] === 'function') {
      try { return this.ytPlayer[method](...args); } catch (e) { console.warn(`YT ${method}:`, e); }
    }
    return defaultValue;
  }

  private initYoutubePlayer(): void {
    const setupPlayer = () => {
      this.ytPlayer = new (window as any).YT.Player('youtube-player-element', {
        height: '100%', width: '100%', videoId: '',
        playerVars: { autoplay: 0, controls: 0, disablekb: 1, fs: 0, rel: 0 },
        events: {
          onReady: (event: any) => {
            this.isReady = true; this.ytPlayer = event.target;
            this.safeCall('setVolume', [this._volume$.value]);
            this.syncPlayerWithServerState();
          },
          onStateChange: (event: any) => {
            this.ytPlayer = event.target;
            this.handlePlayerStateChange(event.data);
          }
        }
      });
    };
    if ((window as any).YT && (window as any).YT.Player) {
      setupPlayer();
    } else {
      const existing = (window as any).onYouTubeIframeAPIReady;
      (window as any).onYouTubeIframeAPIReady = () => { if (existing) existing(); setupPlayer(); };
      const interval = setInterval(() => {
        if ((window as any).YT && (window as any).YT.Player) { clearInterval(interval); setupPlayer(); }
      }, 500);
    }
  }

  private initSoundCloudPlayer(): void {
    const iframe = document.getElementById('soundcloud-player-element') as HTMLIFrameElement;
    if (!iframe) return;
    if ((window as any).SC && (window as any).SC.Widget) {
      this.scWidget = (window as any).SC.Widget(iframe);
      this.scWidget.bind((window as any).SC.Widget.Events.READY, () => {
        this.isScReady = true; this.scWidget.setVolume(this._volume$.value);
        if (this.currentTrack && this.currentTrack.source === 'soundcloud') this.syncSoundCloudPlayer();
      });
      this.scWidget.bind((window as any).SC.Widget.Events.PLAY, () => this.handleScStateChange('playing'));
      this.scWidget.bind((window as any).SC.Widget.Events.PAUSE, () => this.handleScStateChange('paused'));
    } else {
      setTimeout(() => this.initSoundCloudPlayer(), 500);
    }
  }

  private initHtml5Audio(): void {
    this.audioEl = document.getElementById('html5-audio-element') as HTMLAudioElement;
    if (this.audioEl) {
      this.audioEl.volume = this._volume$.value / 100;
      this.audioEl.onplay = () => this.handleHtml5StateChange('playing');
      this.audioEl.onpause = () => this.handleHtml5StateChange('paused');
    }
  }

  private handlePlaybackSync(state: PlaybackState): void {
    this._currentState$.next(state.state);
    this._currentTrack$.next(state.current_track);
    this.lastServerPosition = state.position_ms;
    this.lastUpdatedAt = state.updated_at;
    this.syncPlayerWithServerState();
    this.startProgressTimer();
  }

  private syncPlayerWithServerState(): void {
    const track = this.currentTrack;
    if (!track) { this.stopAllPlayers(); return; }
    const source = track.source || 'youtube';
    if (source === 'youtube') { this.pauseSoundCloud(); this.pauseHtml5Audio(); this.syncYoutubePlayer(); }
    else if (source === 'soundcloud') { this.stopYoutube(); this.pauseHtml5Audio(); this.syncSoundCloudPlayer(); }
    else { this.stopYoutube(); this.pauseSoundCloud(); this.syncHtml5Audio(); }
  }

  private stopAllPlayers(): void { this.stopYoutube(); this.pauseSoundCloud(); this.pauseHtml5Audio(); }
  private stopYoutube(): void { if (this.ytPlayer && this.isReady) this.safeCall('stopVideo'); }
  private pauseSoundCloud(): void { if (this.scWidget && this.isScReady) this.scWidget.pause(); }
  private pauseHtml5Audio(): void { this.audioEl?.pause(); }

  private syncYoutubePlayer(): void {
    if (!this.ytPlayer || !this.isReady) return;
    const track = this.currentTrack!;
    const videoId = this.extractYoutubeId(track.source_url);
    if (!videoId) return;
    let currentYTVideo = '';
    try { const vd = this.safeCall('getVideoData'); if (vd) currentYTVideo = vd.video_id; } catch (e) {}

    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    const targetPosSec = targetPosMS / 1000;
    this.isSyncingLocal = true;

    if (currentYTVideo !== videoId) {
      this.safeCall('cueVideoById', [{ videoId, startSeconds: targetPosSec > 0 ? targetPosSec : 0 }]);
    }
    setTimeout(() => {
      try {
        const playerState = this.safeCall('getPlayerState', [], -1);
        const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
        if (this.currentState === 'playing') {
          const localTimeSec = this.safeCall('getCurrentTime', [], 0);
          if (Math.abs(localTimeSec - targetPosSec) > 1.5) this.safeCall('seekTo', [targetPosSec, true]);
          if (playerState !== ytPlaying) this.safeCall('playVideo');
        } else if (this.currentState === 'paused') {
          this.safeCall('seekTo', [targetPosSec, true]); this.safeCall('pauseVideo');
        } else { this.safeCall('stopVideo'); }
      } catch (e) { console.warn('YT adjust:', e); }
      setTimeout(() => { this.isSyncingLocal = false; }, 500);
    }, currentYTVideo !== videoId ? 800 : 50);
  }

  private syncSoundCloudPlayer(): void {
    if (!this.scWidget || !this.isScReady || !this.currentTrack) return;
    const track = this.currentTrack;
    const currentURL = track.source_url;
    this.scWidget.getCurrentSound((sound: any) => {
      const widgetURL = sound ? sound.permalink_url : '';
      const serverTimeEst = this.playbackWs.getEstimatedServerTime();
      let targetPosMS = this.lastServerPosition;
      if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
      this.isSyncingLocal = true;
      const performSync = () => {
        this.scWidget.seekTo(targetPosMS);
        if (this.currentState === 'playing') this.scWidget.play();
        else if (this.currentState === 'paused') this.scWidget.pause();
        setTimeout(() => { this.isSyncingLocal = false; }, 500);
      };
      if (!widgetURL || !this.isUrlsSame(widgetURL, currentURL)) {
        this.scWidget.load(currentURL, { auto_play: this.currentState === 'playing', callback: () => performSync() });
      } else { performSync(); }
    });
  }

  private syncHtml5Audio(): void {
    if (!this.audioEl || !this.currentTrack) return;
    const track = this.currentTrack;
    const serverTimeEst = this.playbackWs.getEstimatedServerTime();
    let targetPosMS = this.lastServerPosition;
    if (this.currentState === 'playing') targetPosMS += (serverTimeEst - this.lastUpdatedAt);
    const targetPosSec = targetPosMS / 1000;
    this.isSyncingLocal = true;
    if (this.audioEl.src !== track.source_url) { this.audioEl.src = track.source_url; this.audioEl.load(); }
    const drift = Math.abs(this.audioEl.currentTime - targetPosSec);
    if (drift > 1.5) this.audioEl.currentTime = targetPosSec;
    if (this.currentState === 'playing') this.audioEl.play().catch(err => console.warn('HTML5 play:', err));
    else this.audioEl.pause();
    setTimeout(() => { this.isSyncingLocal = false; }, 500);
  }

  private handleScStateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack) return;
    this.scWidget.getPosition((posMS: number) => {
      let action: 'play' | 'pause' | '' = '';
      if (state === 'playing' && this.currentState !== 'playing') action = 'play';
      else if (state === 'paused' && this.currentState === 'playing') action = 'pause';
      if (action) this.playbackWs.sendControlCommand(action, Math.floor(posMS));
    });
  }

  private handleHtml5StateChange(state: 'playing' | 'paused'): void {
    if (this.isSyncingLocal || !this.currentTrack || !this.audioEl) return;
    const posMS = Math.floor(this.audioEl.currentTime * 1000);
    let action: 'play' | 'pause' | '' = '';
    if (state === 'playing' && this.currentState !== 'playing') action = 'play';
    else if (state === 'paused' && this.currentState === 'playing') action = 'pause';
    if (action) this.playbackWs.sendControlCommand(action, posMS);
  }

  private handlePlayerStateChange(state: number): void {
    if (this.isSyncingLocal || !this.currentTrack) return;
    let action: 'play' | 'pause' | '' = '';
    let currentPosMS = 0;
    try {
      currentPosMS = Math.floor((this.safeCall('getCurrentTime', [], 0) || 0) * 1000);
      const ytPlaying = (window as any).YT?.PlayerState?.PLAYING ?? 1;
      const ytPaused = (window as any).YT?.PlayerState?.PAUSED ?? 2;
      if (state === ytPlaying && this.currentState !== 'playing') action = 'play';
      else if (state === ytPaused && this.currentState === 'playing') action = 'pause';
    } catch (e) {}
    if (action) this.playbackWs.sendControlCommand(action, currentPosMS);
  }

  private isUrlsSame(u1: string, u2: string): boolean {
    const clean = (u: string) => u.replace('https://', '').replace('http://', '').replace('www.', '').split('?')[0];
    return clean(u1) === clean(u2);
  }

  private startProgressTimer(): void {
    if (this.progressTimer) clearInterval(this.progressTimer);
    this.progressTimer = setInterval(() => {
      const track = this.currentTrack;
      if (!track) { this._progressMs$.next(0); this._progressPct$.next(0); return; }
      const ms = this.getLocalProgress();
      this._progressMs$.next(ms);
      this._progressPct$.next((ms / track.duration_ms) * 100);
    }, 200);
  }

  /* ── Public commands (used by UI components) ────────────── */
  public getLocalProgress(): number {
    if (this.currentState !== 'playing' || this.lastUpdatedAt === 0) return this.lastServerPosition;
    const elapsed = this.playbackWs.getEstimatedServerTime() - this.lastUpdatedAt;
    let pos = this.lastServerPosition + elapsed;
    if (this.currentTrack && pos > this.currentTrack.duration_ms) pos = this.currentTrack.duration_ms;
    return pos;
  }

  public togglePlayPause(): void {
    if (!this.currentTrack) return;
    const action = this.currentState === 'playing' ? 'pause' : 'play';
    this.playbackWs.sendControlCommand(action, this.getLocalProgress());
  }

  public seekMsFromFraction(fraction: number): number {
    return Math.max(0, Math.min(1, fraction)) * (this.currentTrack?.duration_ms ?? 0);
  }

  public seekToMs(ms: number): void {
    if (!this.currentTrack) return;
    this.playbackWs.sendControlCommand('seek', ms);
  }

  public seekFromFraction(fraction: number): void {
    this.seekToMs(this.seekMsFromFraction(fraction));
  }

  public setVolumeFromFraction(fraction: number): void {
    const v = Math.round(Math.max(0, Math.min(1, fraction)) * 100);
    this._volume$.next(v);
    const source = this.currentTrack?.source || 'youtube';
    if (source === 'youtube') this.safeCall('setVolume', [v]);
    else if (source === 'soundcloud' && this.scWidget && this.isScReady) this.scWidget.setVolume(v);
    else if (this.audioEl) this.audioEl.volume = v / 100;
  }

  public formatTime(ms: number): string {
    const secTotal = Math.floor(ms / 1000);
    return `${String(Math.floor(secTotal / 60)).padStart(2, '0')}:${String(secTotal % 60).padStart(2, '0')}`;
  }

  private extractYoutubeId(urlStr: string): string | null {
    const reg = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
    const m = urlStr.match(reg);
    return (m && m[2].length === 11) ? m[2] : null;
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    if (this.progressTimer) clearInterval(this.progressTimer);
    if (this.ytPlayer && this.ytPlayer.destroy) this.ytPlayer.destroy();
    if (this.audioEl) { this.audioEl.pause(); this.audioEl.src = ''; }
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/player-engine.service.spec.ts"`
Expected: PASS (3 specs).

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/components/player-engine/
git commit -m "feat(room): extract PlayerEngine service from legacy player component"
```

---

## Task 8: Seekbar component

**Files:**
- Create: `web-client/src/app/features/room/components/seekbar/seekbar.component.ts`
- Create: `web-client/src/app/features/room/components/seekbar/seekbar.component.spec.ts`

- [ ] **Step 1: Write the failing test**

```typescript
import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { SeekbarComponent } from './seekbar.component';

describe('SeekbarComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('renders progress fill width from pct input', () => {
    const fixture = TestBed.createComponent(SeekbarComponent);
    const cmp = fixture.componentInstance;
    cmp.pct = 40;
    fixture.detectChanges();
    const fill = fixture.debugElement.query(By.css('.seek-fill')).nativeElement as HTMLElement;
    expect(fill.style.width).toBe('40%');
  });

  it('emits seek with fraction on click', () => {
    const fixture = TestBed.createComponent(SeekbarComponent);
    const cmp = fixture.componentInstance;
    cmp.pct = 0;
    fixture.detectChanges();
    let got = -1;
    cmp.seek.subscribe(f => (got = f));

    const bar = fixture.debugElement.query(By.css('.seek-bar')).nativeElement as HTMLElement;
    spyOnProperty(bar, 'getBoundingClientRect').and.returnValue({ left: 0, width: 200, right: 200, top: 0, bottom: 0, height: 0, x: 0, y: 0, toJSON: () => ({}) } as DOMRect);
    bar.dispatchEvent(new MouseEvent('click', { clientX: 100 }));

    expect(got).toBeCloseTo(0.5, 1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/seekbar.component.spec.ts"`
Expected: FAIL.

- [ ] **Step 3: Implement SeekbarComponent**

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Progress/seek bar. Emits fraction [0..1] on click. */
@Component({
  selector: 'app-room-seekbar',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="seek-bar" (click)="onClick($event)" role="slider"
         [attr.aria-valuenow]="Math.round(pct)" aria-valuemin="0" aria-valuemax="100" aria-label="Seek">
      <div class="seek-track">
        <div class="seek-fill" [style.width.%]="pct"></div>
        <div class="seek-thumb" [style.left.%]="pct"></div>
      </div>
    </div>
  `,
  styles: [`
    .seek-bar { cursor: pointer; padding: 8px 0; }
    .seek-track { height: 4px; border-radius: var(--radius-full); background: var(--bg-elevated); position: relative; transition: var(--transition-fast); }
    .seek-bar:hover .seek-track { height: 6px; }
    .seek-fill { height: 100%; border-radius: inherit; background: var(--accent-gradient); }
    .seek-thumb { position: absolute; top: 50%; width: 12px; height: 12px; transform: translate(-50%, -50%); border-radius: 50%; background: var(--accent-primary); box-shadow: var(--accent-glow); opacity: 0; transition: var(--transition-fast); }
    .seek-bar:hover .seek-thumb { opacity: 1; }
  `]
})
export class SeekbarComponent {
  @Input() pct = 0;
  @Output() seek = new EventEmitter<number>();

  protected Math = Math;

  onClick(event: MouseEvent): void {
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    const fraction = (event.clientX - rect.left) / rect.width;
    this.seek.emit(fraction);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx ng test --watch=false --browsers=ChromeHeadless --include="**/seekbar.component.spec.ts"`
Expected: PASS (2 specs).

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/components/seekbar/
git commit -m "feat(room): add Seekbar component"
```

---

## Task 9: VolumeControl component

**Files:**
- Create: `web-client/src/app/features/room/components/volume-control/volume-control.component.ts`

- [ ] **Step 1: Implement VolumeControlComponent**

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';

/** Volume slider. Emits fraction [0..1] on click. */
@Component({
  selector: 'app-room-volume-control',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="volume">
      <svg class="vol-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M11 5L6 9H2v6h4l5 4V5z"/>
        <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
      </svg>
      <div class="volume-bar" (click)="onClick($event)" role="slider"
           [attr.aria-valuenow]="volumePct" aria-valuemin="0" aria-valuemax="100" aria-label="Volume">
        <div class="volume-fill" [style.width.%]="volumePct"></div>
      </div>
    </div>
  `,
  styles: [`
    .volume { display: flex; align-items: center; gap: 8px; }
    .vol-icon { width: 18px; height: 18px; color: var(--text-secondary); }
    .volume-bar { width: 90px; height: 4px; border-radius: var(--radius-full); background: var(--bg-elevated); cursor: pointer; position: relative; }
    .volume-bar:hover { height: 6px; }
    .volume-fill { height: 100%; border-radius: inherit; background: var(--accent-primary); }
  `]
})
export class VolumeControlComponent {
  @Input() volumePct = 80;
  @Output() volume = new EventEmitter<number>();

  onClick(event: MouseEvent): void {
    const bar = event.currentTarget as HTMLElement;
    const rect = bar.getBoundingClientRect();
    this.volume.emit((event.clientX - rect.left) / rect.width);
  }
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/app/features/room/components/volume-control/
git commit -m "feat(room): add VolumeControl component"
```

---

## Task 10: TransportControls component

**Files:**
- Create: `web-client/src/app/features/room/components/transport-controls/transport-controls.component.ts`

- [ ] **Step 1: Implement TransportControlsComponent**

Uses `IconButtonComponent` + a large gradient play/pause. Emits discrete events.

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';

@Component({
  selector: 'app-room-transport',
  standalone: true,
  imports: [CommonModule, IconButtonComponent],
  template: `
    <div class="transport">
      <app-icon-button label="Phát trước" (clicked)="prev.emit()">
        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 6h2v12H6zM9.5 12l8.5 6V6z"/></svg>
      </app-icon-button>

      <button class="play-main" type="button" [attr.aria-label]="isPlaying ? 'Tạm dừng' : 'Phát'" (click)="playPause.emit()">
        @if (isPlaying) {
          <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
        } @else {
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>
        }
      </button>

      <app-icon-button label="Phát tiếp" (clicked)="next.emit()">
        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M16 6h2v12h-2zM6 18l8.5-6L6 6z"/></svg>
      </app-icon-button>
    </div>
  `,
  styles: [`
    .transport { display: flex; align-items: center; gap: 14px; }
    .play-main {
      width: 46px; height: 46px; border: none; border-radius: 50%;
      background: var(--accent-gradient); color: #05121f; cursor: pointer;
      display: flex; align-items: center; justify-content: center;
      box-shadow: var(--shadow-accent); transition: var(--transition-fast);
    }
    .play-main:hover { transform: scale(1.06); }
    .play-main:active { transform: scale(0.96); }
    .play-main svg { width: 20px; height: 20px; }
  `]
})
export class TransportControlsComponent {
  @Input() isPlaying = false;
  @Output() playPause = new EventEmitter<void>();
  @Output() prev = new EventEmitter<void>();
  @Output() next = new EventEmitter<void>();
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/app/features/room/components/transport-controls/
git commit -m "feat(room): add TransportControls component"
```

---

## Task 11: NowPlayingInfo component

Reusable metadata block used in both Stage and PlayerBar thumbnail area.

**Files:**
- Create: `web-client/src/app/features/room/components/now-playing-info/now-playing-info.component.ts`

- [ ] **Step 1: Implement NowPlayingInfoComponent**

```typescript
import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Track } from '../../../../core/services/websocket/playback-ws.service';

@Component({
  selector: 'app-room-now-playing',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="now-playing" [class.compact]="compact">
      @if (track?.thumbnail_url) {
        <img class="np-thumb" [src]="track!.thumbnail_url" [attr.alt]="track?.title">
      } @else {
        <div class="np-thumb np-thumb-ph" aria-hidden="true">🎜</div>
      }
      <div class="np-meta">
        <span class="np-title" [title]="track ? track!.title : ''">{{ track ? track!.title : 'Chưa phát bài hát nào' }}</span>
        <span class="np-artist">{{ track ? track!.artist : 'Hàng đợi đang trống' }}</span>
      </div>
    </div>
  `,
  styles: [`
    .now-playing { display: flex; align-items: center; gap: 10px; min-width: 0; }
    .np-thumb { width: 44px; height: 44px; border-radius: var(--radius-md); object-fit: cover; background: var(--bg-elevated); flex-shrink: 0; }
    .np-thumb-ph { display: flex; align-items: center; justify-content: center; color: var(--text-muted); }
    .np-meta { display: flex; flex-direction: column; min-width: 0; }
    .np-title { font-weight: 600; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .np-artist { font-size: var(--text-xs-size); color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  `]
})
export class NowPlayingInfoComponent {
  @Input() track: Track | null = null;
  @Input() compact = false;
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/app/features/room/components/now-playing-info/
git commit -m "feat(room): add NowPlayingInfo component"
```

---

## Task 12: ArtworkVisualizer component (pseudo-visualizer + halo glow)

24 bars around the artwork. Heights driven by a lightweight pseudo-visualizer: seeded random heights animated via CSS keyframes, re-seeded on track change / play toggle. Halo glow pulsing when playing. All disabled under `prefers-reduced-motion`.

**Files:**
- Create: `web-client/src/app/features/room/components/artwork-visualizer/artwork-visualizer.component.ts`

- [ ] **Step 1: Implement ArtworkVisualizerComponent**

```typescript
import { Component, Input, OnChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Track } from '../../../../core/services/websocket/playback-ws.service';

@Component({
  selector: 'app-room-artwork-visualizer',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="art-stage" [class.playing]="isPlaying">
      <div class="halo" aria-hidden="true"></div>
      <div class="bars" aria-hidden="true">
        @for (h of barHeights; track $index) {
          <span class="bar" [style.height.%]="h" [style.animation-delay.ms]="$index * 60"></span>
        }
      </div>
      <div class="artwork">
        @if (track?.thumbnail_url) {
          <img [src]="track!.thumbnail_url" [attr.alt]="track?.title">
        } @else {
          <span class="placeholder">🎜</span>
        }
      </div>
    </div>
  `,
  styles: [`
    .art-stage { position: relative; width: 280px; height: 280px; display: flex; align-items: center; justify-content: center; }
    .artwork {
      width: 180px; height: 180px; border-radius: var(--radius-xl); overflow: hidden;
      box-shadow: var(--shadow-lg); background: var(--bg-elevated);
      display: flex; align-items: center; justify-content: center; position: relative; z-index: 2;
    }
    .artwork img { width: 100%; height: 100%; object-fit: cover; }
    .placeholder { font-size: 48px; color: var(--text-muted); }
    .halo {
      position: absolute; inset: 0; border-radius: 50%;
      background: radial-gradient(circle, var(--accent-glow-color) 0%, transparent 60%);
      opacity: 0; transition: var(--transition-slow); z-index: 1;
    }
    .art-stage.playing .halo { opacity: 1; animation: halo-pulse 2.4s var(--ease-out-expo) infinite; }
    .bars { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; gap: 4px; z-index: 1; }
    .bar { width: 5px; height: 20%; border-radius: var(--radius-full); background: var(--accent-gradient); opacity: 0.55; }
    .art-stage.playing .bar { animation: bar-bounce 0.9s var(--ease-out-expo) infinite alternate; }
    @keyframes halo-pulse { 0%,100% { transform: scale(1); opacity: 0.6; } 50% { transform: scale(1.08); opacity: 1; } }
    @keyframes bar-bounce { from { transform: scaleY(0.6); } to { transform: scaleY(1.6); } }
    @media (prefers-reduced-motion: reduce) {
      .art-stage.playing .halo { animation: none; opacity: 0.6; }
      .art-stage.playing .bar { animation: none; }
    }
  `]
})
export class ArtworkVisualizerComponent implements OnChanges {
  @Input() track: Track | null = null;
  @Input() isPlaying = false;

  public barHeights = this.seedBars();

  ngOnChanges(): void {
    // Re-seed on track/play change so the visualizer "restarts".
    this.barHeights = this.seedBars();
  }

  private seedBars(): number[] {
    return Array.from({ length: 24 }, () => 20 + Math.round(Math.random() * 70));
  }
}
```

- [ ] **Step 2: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web-client/src/app/features/room/components/artwork-visualizer/
git commit -m "feat(room): add ArtworkVisualizer component with pseudo-visualizer + halo"
```

---

## Task 13: PlayerBar component (full-width bottom control bar)

Hosts the hidden YouTube/SoundCloud/HTML5 player elements (by id), NowPlayingInfo, TransportControls, Seekbar, VolumeControl, and queue/chat toggle buttons. Wires UI to `PlayerEngineService`.

**Files:**
- Create: `web-client/src/app/features/room/components/player-bar/player-bar.component.ts`
- Create: `web-client/src/app/features/room/components/player-bar/player-bar.component.html`
- Create: `web-client/src/app/features/room/components/player-bar/player-bar.component.css`

- [ ] **Step 1: Implement PlayerBarComponent (TS)**

```typescript
import { AfterViewInit, Component, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { RoomUiStateService } from '../../room-ui-state.service';
import { NowPlayingInfoComponent } from '../now-playing-info/now-playing-info.component';
import { TransportControlsComponent } from '../transport-controls/transport-controls.component';
import { SeekbarComponent } from '../seekbar/seekbar.component';
import { VolumeControlComponent } from '../volume-control/volume-control.component';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';

@Component({
  selector: 'app-room-player-bar',
  standalone: true,
  imports: [
    CommonModule, NowPlayingInfoComponent, TransportControlsComponent,
    SeekbarComponent, VolumeControlComponent, IconButtonComponent
  ],
  templateUrl: './player-bar.component.html',
  styleUrl: './player-bar.component.css'
})
export class PlayerBarComponent implements AfterViewInit, OnDestroy {
  public engine = inject(PlayerEngineService);
  public uiState = inject(RoomUiStateService);

  public isPlaying = false;
  public pct = 0;
  public progressMs = 0;
  public volumePct = 80;
  public queueOpen = true;
  public chatOpen = false;

  private subs: Subscription[] = [];

  ngAfterViewInit(): void {
    this.engine.loadApis();
    // Defer init so the host iframe/audio elements exist in the DOM.
    setTimeout(() => this.engine.initPlayers(), 0);

    this.subs.push(
      this.engine.currentState$.subscribe(s => (this.isPlaying = s === 'playing')),
      this.engine.progressPct$.subscribe(p => (this.pct = p)),
      this.engine.progressMs$.subscribe(ms => (this.progressMs = ms)),
      this.engine.volume$.subscribe(v => (this.volumePct = v)),
      this.uiState.changes$.subscribe(st => {
        this.queueOpen = st.isQueueOpen;
        this.chatOpen = st.isChatOpen;
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
  }

  onPlayPause(): void { this.engine.togglePlayPause(); }
  onSeek(fraction: number): void { this.engine.seekFromFraction(fraction); }
  onVolume(fraction: number): void { this.engine.setVolumeFromFraction(fraction); }
  onPrev(): void { /* playback API has no prev; left as no-op hook for future */ }
  onNext(): void { /* no-op hook for future */ }
  toggleQueue(): void { this.uiState.toggleQueue(); }
  toggleChat(): void { this.uiState.toggleChat(); }
}
```

- [ ] **Step 2: Write player-bar.component.html**

```html
<!-- Hidden media hosts (by id, used by PlayerEngineService) -->
<div class="media-host" aria-hidden="true">
  <div id="youtube-player-element"></div>
  <iframe id="soundcloud-player-element" width="1" height="1" scrolling="no" frameborder="no" allow="autoplay" src=""></iframe>
</div>
<audio id="html5-audio-element" style="display:none;"></audio>

<div class="player-bar">
  <div class="pb-left">
    <app-room-now-playing [track]="engine.currentTrack"></app-room-now-playing>
  </div>

  <div class="pb-center">
    <app-room-transport [isPlaying]="isPlaying"
                        (playPause)="onPlayPause()"
                        (prev)="onPrev()"
                        (next)="onNext()"></app-room-transport>
    <div class="pb-seek">
      <span class="pb-time">{{ engine.formatTime(progressMs) }}</span>
      <app-room-seekbar [pct]="pct" (seek)="onSeek($event)"></app-room-seekbar>
      <span class="pb-time">{{ engine.formatTime(engine.currentTrack ? engine.currentTrack.duration_ms : 0) }}</span>
    </div>
  </div>

  <div class="pb-right">
    <app-room-volume-control [volumePct]="volumePct" (volume)="onVolume($event)"></app-room-volume-control>
    <app-icon-button [label]="queueOpen ? 'Ẩn hàng đợi' : 'Mở hàng đợi'" [variant]="queueOpen ? 'active' : 'default'" (clicked)="toggleQueue()">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
    </app-icon-button>
    <app-icon-button [label]="chatOpen ? 'Đóng chat' : 'Mở chat'" [variant]="chatOpen ? 'active' : 'default'" (clicked)="toggleChat()">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
    </app-icon-button>
  </div>
</div>
```

- [ ] **Step 3: Write player-bar.component.css**

```css
:host { display: block; }
.media-host { position: absolute; width: 1px; height: 1px; overflow: hidden; left: -9999px; }
.player-bar {
  display: flex; align-items: center; gap: 16px;
  height: 80px; padding: 0 20px;
  background: var(--bg-glass); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
  border-top: 1px solid var(--border-color);
}
.pb-left { flex: 0 0 220px; min-width: 0; }
.pb-center { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px; min-width: 0; }
.pb-seek { display: flex; align-items: center; gap: 10px; width: 100%; max-width: 520px; }
.pb-time { font-family: var(--font-mono); font-size: var(--text-xs-size); color: var(--text-secondary); min-width: 40px; text-align: center; }
.pb-seek app-room-seekbar { flex: 1; }
.pb-right { flex: 0 0 auto; display: flex; align-items: center; gap: 6px; }
@media (max-width: 768px) {
  .pb-left { display: none; }
  .pb-right app-room-volume-control { display: none; }
}
```

- [ ] **Step 4: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/components/player-bar/
git commit -m "feat(room): add PlayerBar component (full-width bottom controls)"
```

---

## Task 14: VoicePill + VoicePills components

`VoicePill` renders a single participant; `VoicePills` subscribes to `VoiceService.participants$` + `activeSpeakers$` + room members and renders floating pills (audio-only representation; video/screenshare is handled by Stage in Task 15). Port the participant-merge/sort logic from the existing `VoiceGridComponent` (syncParticipants, participantWeight, roleWeight).

**Files:**
- Create: `web-client/src/app/features/room/components/voice-pills/voice-pill.component.ts`
- Create: `web-client/src/app/features/room/components/voice-pills/voice-pills.component.ts`
- Create: `web-client/src/app/features/room/components/voice-pills/voice-pills.component.css`

- [ ] **Step 1: Implement VoicePillComponent**

```typescript
import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

export interface PillParticipant {
  sid: string;
  display_name: string;
  avatar_url?: string;
  isSpeaking: boolean;
  isMuted: boolean;
  isCurrentUser: boolean;
}

@Component({
  selector: 'app-room-voice-pill',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="voice-pill" [class.speaking]="p.isSpeaking" [class.muted]="p.isMuted">
      <div class="vp-avatar">
        @if (p.avatar_url) {
          <img [src]="p.avatar_url" [alt]="p.display_name">
        } @else {
          <span>{{ initials }}</span>
        }
        @if (p.isSpeaking) { <span class="vp-ring" aria-hidden="true"></span> }
      </div>
      <span class="vp-name">{{ p.display_name }}{{ p.isCurrentUser ? ' (Bạn)' : '' }}</span>
      @if (p.isMuted) {
        <svg class="vp-mic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Đã tắt mic"><line x1="1" y1="1" x2="23" y2="23"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"/></svg>
      }
    </div>
  `,
  styles: [`
    .voice-pill { display: flex; align-items: center; gap: 8px; padding: 6px 12px; border-radius: var(--radius-full); background: var(--bg-glass); backdrop-filter: blur(12px); border: 1px solid var(--border-color); transition: var(--transition-normal); }
    .voice-pill.speaking { border-color: var(--accent-primary); box-shadow: var(--accent-glow); transform: scale(1.04); }
    .vp-avatar { position: relative; width: 28px; height: 28px; border-radius: 50%; overflow: hidden; background: var(--bg-elevated); display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: var(--text-primary); }
    .vp-avatar img { width: 100%; height: 100%; object-fit: cover; }
    .vp-ring { position: absolute; inset: -3px; border: 2px solid var(--accent-primary); border-radius: 50%; animation: ring-pulse 1.2s var(--ease-out-expo) infinite; }
    @keyframes ring-pulse { 0%,100% { opacity: 0.7; } 50% { opacity: 1; } }
    .vp-name { font-size: var(--text-sm-size); color: var(--text-primary); white-space: nowrap; }
    .vp-mic { width: 14px; height: 14px; color: var(--danger); }
    @media (prefers-reduced-motion: reduce) { .vp-ring { animation: none; } }
  `]
})
export class VoicePillComponent {
  @Input({ required: true }) p!: PillParticipant;
  get initials(): string {
    return (this.p?.display_name || 'WT').slice(0, 2).toUpperCase();
  }
}
```

- [ ] **Step 2: Implement VoicePillsComponent (port merge/sort from VoiceGrid)**

```typescript
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { VoiceService } from '../../../../core/services/voice.service';
import { StateService } from '../../../../core/services/state.service';
import { VoicePillComponent, PillParticipant } from './voice-pill.component';

@Component({
  selector: 'app-room-voice-pills',
  standalone: true,
  imports: [CommonModule, VoicePillComponent],
  template: `
    @if (pills.length > 0) {
      <div class="voice-pills" role="list">
        @for (p of pills; track p.sid) {
          <app-room-voice-pill role="listitem" [p]="p"></app-room-voice-pill>
        }
      </div>
    }
  `,
  styleUrl: './voice-pills.component.css'
})
export class VoicePillsComponent implements OnInit, OnDestroy {
  private voiceService = inject(VoiceService);
  private state = inject(StateService);

  public pills: PillParticipant[] = [];
  private rawParticipants: any[] = [];
  private members: any[] = [];
  private activeSpeakers: string[] = [];
  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.voiceService.participants$.subscribe(list => { this.rawParticipants = list || []; this.sync(); }),
      this.voiceService.activeSpeakers$.subscribe(s => { this.activeSpeakers = s || []; this.sync(); }),
      this.state.activeRoomMembers$.subscribe(m => { this.members = m || []; this.sync(); })
    );
  }

  ngOnDestroy(): void { this.subs.forEach(s => s.unsubscribe()); }

  private sync(): void {
    const live = this.rawParticipants.map(p => {
      const member = this.members.find(m => m.user_id === p.identity || m.id === p.identity);
      return {
        sid: p.sid,
        display_name: member?.display_name || p.identity || 'Unknown',
        avatar_url: member?.avatar_url || '',
        isSpeaking: this.activeSpeakers.includes(p.sid),
        isMuted: !!p.isMuted,
        isCurrentUser: !!member?.isCurrentUser || !!p.isLocal
      } as PillParticipant;
    });
    const liveIds = new Set(this.rawParticipants.map(p => p.identity).filter(Boolean));
    const silent = this.members
      .filter(m => !liveIds.has(m.user_id))
      .map(m => ({
        sid: `member-${m.user_id}`,
        display_name: m.display_name || m.username || 'Thành viên',
        avatar_url: m.avatar_url || '',
        isSpeaking: false,
        isMuted: false,
        isCurrentUser: !!m.isCurrentUser
      } as PillParticipant));
    this.pills = [...live, ...silent].sort((a, b) => this.weight(b) - this.weight(a));
  }

  private weight(p: PillParticipant): number {
    return (p.isSpeaking ? 1000 : 0) + (p.sid.startsWith('member-') ? 0 : 500) + (p.isCurrentUser ? 1 : 0);
  }
}
```

- [ ] **Step 3: Write voice-pills.component.css**

```css
:host { display: block; }
.voice-pills {
  position: absolute; top: 16px; left: 16px; right: 16px;
  display: flex; flex-wrap: wrap; gap: 8px; z-index: var(--z-overlay);
  pointer-events: none;
}
.voice-pills app-room-voice-pill { pointer-events: auto; }
```

- [ ] **Step 4: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/components/voice-pills/
git commit -m "feat(room): add VoicePills + VoicePill floating components"
```

---

## Task 15: Stage component (center stage)

Shows the ArtworkVisualizer + NowPlayingInfo when in a music mode; switches to a video/screenshare surface when stage mode is `screenshare` or `video`. The video surface hosts the livekit video elements; it is the minimal Stage surface — full video-track attach logic remains in the legacy/voice-grid path until parity (Task 18) and is wired here as a passthrough slot (`<ng-content select="[stage-media]">`).

**Files:**
- Create: `web-client/src/app/features/room/components/stage/stage.component.ts`
- Create: `web-client/src/app/features/room/components/stage/stage.component.css`

- [ ] **Step 1: Implement StageComponent**

```typescript
import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { RoomUiStateService, StageMode } from '../../room-ui-state.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { ArtworkVisualizerComponent } from '../artwork-visualizer/artwork-visualizer.component';
import { NowPlayingInfoComponent } from '../now-playing-info/now-playing-info.component';

@Component({
  selector: 'app-room-stage',
  standalone: true,
  imports: [CommonModule, ArtworkVisualizerComponent, NowPlayingInfoComponent],
  template: `
    <div class="stage">
      @switch (stageMode) {
        @case ('screenshare') {
          <div class="stage-media"><ng-content select="[stage-media]"></ng-content></div>
        }
        @case ('video') {
          <div class="stage-media"><ng-content select="[stage-media]"></ng-content></div>
        }
        @default {
          <div class="stage-music">
            <app-room-artwork-visualizer [track]="engine.currentTrack" [isPlaying]="isPlaying"></app-room-artwork-visualizer>
            <div class="stage-meta">
              <app-room-now-playing [track]="engine.currentTrack"></app-room-now-playing>
            </div>
          </div>
        }
      }
    </div>
  `,
  styleUrl: './stage.component.css'
})
export class StageComponent {
  public engine = inject(PlayerEngineService);
  private uiState = inject(RoomUiStateService);

  public stageMode: StageMode = 'music-only';
  public isPlaying = false;
  private subs: Subscription[] = [];

  constructor() {
    this.subs.push(
      this.uiState.changes$.subscribe(s => (this.stageMode = s.stageMode)),
      this.engine.currentState$.subscribe(s => (this.isPlaying = s === 'playing'))
    );
  }
}
```

- [ ] **Step 2: Write stage.component.css**

```css
:host { display: block; height: 100%; }
.stage { height: 100%; display: flex; align-items: center; justify-content: center; padding: 32px 24px 110px; }
.stage-music { display: flex; flex-direction: column; align-items: center; gap: 24px; }
.stage-meta { min-width: 260px; justify-content: center; }
.stage-media { width: 100%; height: 100%; background: var(--bg-base); border-radius: var(--radius-lg); overflow: hidden; }
@media (max-width: 768px) { .stage { padding: 16px 12px 96px; } }
```

- [ ] **Step 3: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add web-client/src/app/features/room/components/stage/
git commit -m "feat(room): add Stage component (music + video/screenshare modes)"
```

---

## Task 16: Sidebar component (extract from inline)

Extract the 64px voice-control sidebar (currently inline in `room.component.html`) into its own component, re-skinned Cool Ocean. Receives voice state via inputs; emits control events. The Shell wires these to the same handlers ported from the legacy `RoomComponent`.

**Files:**
- Create: `web-client/src/app/features/room/components/sidebar/sidebar.component.ts`
- Create: `web-client/src/app/features/room/components/sidebar/sidebar.component.css`

- [ ] **Step 1: Implement SidebarComponent**

```typescript
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { IconButtonComponent } from '../../../../shared/components/icon-button/icon-button.component';

@Component({
  selector: 'app-room-sidebar',
  standalone: true,
  imports: [CommonModule, IconButtonComponent],
  template: `
    <div class="sidebar">
      <div class="sidebar-top">
        <app-icon-button label="Quay về Lobby" (clicked)="back.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
        </app-icon-button>

        <app-icon-button label="Kênh voice" [variant]="isMicActive ? 'active' : 'default'" [disabled]="isVoiceBusy" (clicked)="voiceChannel.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 18v-6a9 9 0 0 1 18 0v6"/><path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z"/></svg>
        </app-icon-button>

        <app-icon-button [label]="isMuted ? 'Bật microphone' : 'Tắt microphone'" [variant]="isMicActive && !isMuted ? 'active' : 'default'" [disabled]="!isMicActive || isVoiceBusy" (clicked)="muteToggle.emit()">
          @if (isMicActive && isMuted) {
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="1" y1="1" x2="23" y2="23"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"/><path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"/><line x1="12" y1="19" x2="12" y2="23"/></svg>
          } @else {
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>
          }
        </app-icon-button>

        <app-icon-button label="Chia sẻ màn hình" [variant]="isScreenSharing ? 'active' : 'default'" [disabled]="!isMicActive || isScreenShareBusy" (clicked)="screenShareToggle.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><path d="M8 21h8M12 17v4"/></svg>
        </app-icon-button>

        <app-icon-button label="Camera" [variant]="isCameraActive ? 'active' : 'default'" [disabled]="!isMicActive || isCameraBusy" (clicked)="cameraToggle.emit()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M23 7l-7 5 7 5V7z"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
        </app-icon-button>
      </div>

      <div class="sidebar-bottom">
        <div class="sb-avatar" [title]="displayName">
          @if (avatarUrl) {
            <img [src]="avatarUrl" [alt]="displayName">
          } @else {
            <span>{{ initials }}</span>
          }
        </div>
      </div>
    </div>
  `,
  styleUrl: './sidebar.component.css'
})
export class SidebarComponent {
  @Input() isMicActive = false;
  @Input() isMuted = false;
  @Input() isScreenSharing = false;
  @Input() isCameraActive = false;
  @Input() isVoiceBusy = false;
  @Input() isScreenShareBusy = false;
  @Input() isCameraBusy = false;
  @Input() avatarUrl = '';
  @Input() displayName = '';

  @Output() back = new EventEmitter<void>();
  @Output() voiceChannel = new EventEmitter<void>();
  @Output() muteToggle = new EventEmitter<void>();
  @Output() screenShareToggle = new EventEmitter<void>();
  @Output() cameraToggle = new EventEmitter<void>();

  get initials(): string {
    return (this.displayName || 'WT').slice(0, 2).toUpperCase();
  }
}
```

- [ ] **Step 2: Write sidebar.component.css**

```css
:host { display: block; height: 100%; }
.sidebar {
  width: 64px; height: 100%; flex-shrink: 0;
  background: var(--bg-glass); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
  border-right: 1px solid var(--border-color);
  display: flex; flex-direction: column; align-items: center; justify-content: space-between; padding: 16px 0;
}
.sidebar-top, .sidebar-bottom { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.sb-avatar {
  width: 40px; height: 40px; border-radius: 50%; overflow: hidden;
  background: var(--bg-elevated); border: 1px solid var(--border-color);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 700; color: var(--text-primary);
  box-shadow: var(--inner-bezel); cursor: pointer; transition: var(--transition-fast);
}
.sb-avatar:hover { border-color: var(--border-focus); box-shadow: 0 0 0 2px rgba(45, 212, 191, 0.2); }
.sb-avatar img { width: 100%; height: 100%; object-fit: cover; }
@media (max-width: 480px) { .sidebar { width: 48px; } }
```

- [ ] **Step 3: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add web-client/src/app/features/room/components/sidebar/
git commit -m "feat(room): extract Sidebar component (64px voice controls)"
```

---

## Task 17: Adapt Chat + Queue to new shell (overlay dock + toggle panel)

These two components keep their existing `.ts` logic (ChatWs, ApiService, queue CRUD, drag-reorder). Only template/CSS changes wrap them for the new layout: Chat renders inside an overlay dock driven by `RoomUiStateService.isChatOpen`; Queue renders in a collapsible right panel driven by `isQueueOpen`. Chat adds an unread badge hook: when a message arrives and chat is closed, call `uiState.markUnread(1)`.

**Files:**
- Modify: `web-client/src/app/features/room/components/chat/chat.component.ts`
- Modify: `web-client/src/app/features/room/components/chat/chat.component.css`
- Modify: `web-client/src/app/features/room/components/queue/queue.component.css`

- [ ] **Step 1: Add unread tracking to ChatComponent**

In `chat.component.ts`, inject `RoomUiStateService` and increment unread on each incoming message when chat is closed. Add an `isOpen` input + close button emission so the dock can render/hide it.

Add to the imports and class:

```typescript
import { Input } from '@angular/core';
import { RoomUiStateService } from '../../room-ui-state.service';
// inside class:
private uiState = inject(RoomUiStateService);
@Input() isOpen = false;

// in ngOnInit, change the messageReceived$ subscription to:
this.chatWs.messageReceived$.subscribe((msg) => {
  this.enrichMessage(msg);
  this.messages.push(msg);
  if (!this.uiState.uiState.isChatOpen) {
    this.uiState.markUnread(1);
  }
  setTimeout(() => this.scrollToBottom(), 50);
});

public close(): void {
  this.uiState.toggleChat(false);
}
```

(`@Input() isOpen` is bound from the Shell via `uiState.isChatOpen`.)

- [ ] **Step 2: Update chat.component.css for overlay dock styling**

Append (the chat panel root already has `.room-chat-panel`; we add overlay positioning handled by the Shell, so here we just ensure internal layout fills height and uses ocean tokens). Replace any violet `rgba(108,99,255,...)` references with teal equivalents / tokens. Specifically ensure `.chat-text-input` and `.chat-msg-bubble` use `var(--bg-elevated)` and `var(--accent-dim)`.

Because the existing file uses hardcoded violet hex in places, run a targeted replacement: every `rgba(108, 99, 255, X)` → `rgba(45, 212, 191, X)` and every `#6c63ff` → `#2dd4bf` within `chat.component.css`. (Verify with the build in Step 4.)

- [ ] **Step 3: Update queue.component.css similarly (toggle panel)**

Replace violet references (`rgba(108, 99, 255, X)` → `rgba(45, 212, 191, X)`, `#6c63ff` → `#2dd4bf`) in `queue.component.css`. The toggle open/close is controlled by the Shell wrapper, not the component itself.

- [ ] **Step 4: Verify build**

Run: `npx ng build`
Expected: success, no violet hex remaining in these two CSS files.

Verify with:
```bash
git grep -nE "rgba\(108, 99, 255|6c63ff|7c74ff" -- web-client/src/app/features/room/components/chat web-client/src/app/features/room/components/queue
```
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add web-client/src/app/features/room/components/chat/ web-client/src/app/features/room/components/queue/
git commit -m "feat(room): adapt Chat (overlay + unread) and Queue to Cool Ocean"
```

---

## Task 18: Rename legacy Room + build new RoomShell

Rename the old `RoomComponent` to `RoomLegacyComponent` (keep as fallback), then rebuild `room.component` as the new `RoomShell` composing all new components.

**Files:**
- Rename: `web-client/src/app/features/room/room.component.ts` → `room-legacy.component.ts` (class `RoomLegacyComponent`, selector `app-room-legacy`), plus its `.html`/`.css` and the spec.
- Rebuild: `web-client/src/app/features/room/room.component.{ts,html,css}`

- [ ] **Step 1: Rename legacy component**

`git mv web-client/src/app/features/room/room.component.ts web-client/src/app/features/room/room-legacy.component.ts` (and likewise for `.html`, `.css`, `.spec.ts`).

In `room-legacy.component.ts`: change `selector: 'app-room'` → `'app-room-legacy'`, class `RoomComponent` → `RoomLegacyComponent`. Update its own relative imports (still point to `./components/...`, unchanged). Do NOT register it in routes (it is only a fallback reachable by temporary local edit if needed).

- [ ] **Step 2: Implement new RoomShellComponent**

Port the room orchestration logic from the legacy component (`loadRoomDetails`, `connectWebSockets`, voice handlers `onVoiceChannelClick/onMuteToggleClick/onScreenShareToggleClick/onCameraToggleClick`, `onBackToLobbyClick`, presence heartbeat, members polling, `disconnectAll`). The new shell composes Sidebar + Stage + VoicePills + Chat + Queue + PlayerBar and derives `stageMode` from voice/screenshare state.

```typescript
import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { ApiService } from '../../core/services/api.service';
import { StateService } from '../../core/services/state.service';
import { ChatWsService } from '../../core/services/websocket/chat-ws.service';
import { PlaybackWsService } from '../../core/services/websocket/playback-ws.service';
import { VoiceService } from '../../core/services/voice.service';
import { ToastService } from '../../shared/services/toast.service';
import { RoomUiStateService, StageMode } from './room-ui-state.service';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { StageComponent } from './components/stage/stage.component';
import { VoicePillsComponent } from './components/voice-pills/voice-pills.component';
import { ChatComponent } from './components/chat/chat.component';
import { QueueComponent } from './components/queue/queue.component';
import { PlayerBarComponent } from './components/player-bar/player-bar.component';

@Component({
  selector: 'app-room',
  standalone: true,
  imports: [CommonModule, SidebarComponent, StageComponent, VoicePillsComponent, ChatComponent, QueueComponent, PlayerBarComponent],
  templateUrl: './room.component.html',
  styleUrl: './room.component.css'
})
export class RoomComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(ApiService);
  public state = inject(StateService);
  private chatWs = inject(ChatWsService);
  private playbackWs = inject(PlaybackWsService);
  public voiceService = inject(VoiceService);
  private toast = inject(ToastService);
  private uiState = inject(RoomUiStateService);

  public roomId = '';
  public isMicActive = false;
  public isMuted = false;
  public isScreenSharing = false;
  public isCameraActive = false;
  public isVoiceBusy = false;
  public isScreenShareBusy = false;
  public isCameraBusy = false;
  public isLeavingRoom = false;

  private subs: Subscription[] = [];
  private presenceTimer: any = null;
  private membersPollTimer: any = null;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('room_id');
    if (!id) {
      this.toast.error('Phòng không hợp lệ.');
      this.router.navigate(['/dashboard']);
      return;
    }
    this.roomId = id;
    this.loadRoomDetails();
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    this.disconnectAll();
  }

  private loadRoomDetails(): void {
    this.api.room.get(this.roomId).subscribe({
      next: (room) => {
        this.state.activeRoom$.next(room);
        this.loadRoomMembers();
        this.connectWebSockets();
        setTimeout(() => this.onVoiceChannelClick(), 800);
      },
      error: (err) => {
        this.toast.error('Không thể tải phòng: ' + err.message);
        this.router.navigate(['/dashboard']);
      }
    });
  }

  private connectWebSockets(): void {
    const token = this.state.accessToken;
    if (!token) { this.toast.error('Thiếu token.'); this.router.navigate(['/auth']); return; }
    this.chatWs.connect(this.roomId, token);
    this.playbackWs.connect(this.roomId, token);
    this.startPresenceHeartbeat();
    this.startMembersPolling();

    this.subs.push(
      this.voiceService.connected$.subscribe(c => {
        this.isMicActive = c;
        if (!c) { this.isMuted = false; this.isScreenSharing = false; this.isCameraActive = false; this.isVoiceBusy = false; }
        this.recomputeStageMode();
      }),
      this.voiceService.isMuted$.subscribe(m => (this.isMuted = m)),
      this.voiceService.isScreenSharing$.subscribe(s => { this.isScreenSharing = s; this.recomputeStageMode(); }),
      this.voiceService.isCameraActive$.subscribe(a => { this.isCameraActive = a; this.recomputeStageMode(); }),
      this.voiceService.participants$.subscribe(() => this.recomputeStageMode())
    );
  }

  private recomputeStageMode(): void {
    let mode: StageMode = 'music-only';
    if (this.voiceService.participants$.value.some(p => p.isScreenSharing)) mode = 'screenshare';
    else if (this.voiceService.participants$.value.some(p => p.isCameraOn)) mode = 'video';
    else if (this.isMicActive) mode = 'music-voice';
    this.uiState.setStageMode(mode);
  }

  public onVoiceChannelClick(): Promise<void> {
    if (this.isVoiceBusy) return Promise.resolve();
    if (this.isMicActive) {
      this.voiceService.disconnect();
      this.toast.success('Đã rời kênh voice.');
      return Promise.resolve();
    }
    this.isVoiceBusy = true;
    this.toast.info('Đang kết nối voice...');
    this.api.voice.getToken(this.roomId).subscribe({
      next: async (res) => {
        try { await this.voiceService.connect(res.livekit_url, res.token); this.toast.success('Đã tham gia voice.'); }
        catch (err: any) { this.toast.error('Lỗi voice: ' + err.message); }
        finally { this.isVoiceBusy = false; }
      },
      error: (err) => { this.isVoiceBusy = false; this.toast.error('Không xin được token voice: ' + err.message); }
    });
    return Promise.resolve();
  }

  public async onMuteToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isVoiceBusy) return;
    try {
      const target = !this.isMuted;
      await this.voiceService.setMute(target);
      this.toast.info(target ? 'Đã tắt mic.' : 'Đã bật mic.');
    } catch (err: any) { this.toast.error('Lỗi mic: ' + err.message); }
  }

  public async onScreenShareToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isScreenShareBusy) return;
    try {
      this.isScreenShareBusy = true;
      const target = !this.isScreenSharing;
      await this.voiceService.setScreenShare(target);
      this.toast.success(target ? 'Đã chia sẻ màn hình.' : 'Đã dừng chia sẻ.');
    } catch (err: any) { this.toast.error('Lỗi chia sẻ: ' + err.message); }
    finally { this.isScreenShareBusy = false; }
  }

  public async onCameraToggleClick(): Promise<void> {
    if (!this.isMicActive || this.isCameraBusy) return;
    try {
      this.isCameraBusy = true;
      const target = !this.isCameraActive;
      await this.voiceService.setCamera(target);
      this.toast.info(target ? 'Đã bật camera.' : 'Đã tắt camera.');
    } catch (err: any) { this.toast.error('Lỗi camera: ' + err.message); }
    finally { this.isCameraBusy = false; }
  }

  public async onBackToLobbyClick(): Promise<void> {
    if (this.isLeavingRoom) return;
    this.isLeavingRoom = true;
    try { await this.api.room.leave(this.roomId).toPromise(); }
    catch (err: any) { console.warn('Leave failed:', err); this.toast.info('Đã thoát phòng.'); }
    finally {
      this.disconnectAll();
      this.router.navigate(['/dashboard']);
      this.isLeavingRoom = false;
    }
  }

  // loadRoomMembers / presence / polling / disconnectAll are identical to the
  // legacy implementation; copy them verbatim from room-legacy.component.ts.
  // (Kept out of this snippet for brevity — see Step 3 note.)

  private async loadRoomMembers(): Promise<void> { /* verbatim copy from legacy */ }
  private startPresenceHeartbeat(): void { /* verbatim */ }
  private stopPresenceHeartbeat(): void { /* verbatim */ }
  private startMembersPolling(): void { /* verbatim */ }
  private stopMembersPolling(): void { /* verbatim */ }
  private disconnectAll(): void { /* verbatim */ }

  get currentUserAvatar(): string { return this.state.user?.avatar_url || ''; }
  get currentUserDisplayName(): string {
    const u = this.state.user;
    return u?.display_name || u?.username || 'WT';
  }
  get isChatOpen(): boolean { return this.uiState.uiState.isChatOpen; }
  get isQueueOpen(): boolean { return this.uiState.uiState.isQueueOpen; }
}
```

- [ ] **Step 3: Copy the verbatim helper bodies**

Open `room-legacy.component.ts` and copy the exact bodies of `loadRoomMembers`, `startPresenceHeartbeat`, `stopPresenceHeartbeat`, `startMembersPolling`, `stopMembersPolling`, `disconnectAll` into the new `room.component.ts`, replacing the placeholder comments. These are byte-for-byte copies (logic unchanged). Verify field names match: `presenceTimer`, `membersPollTimer`, `state`, `chatWs`, `playbackWs`, `voiceService` all exist.

- [ ] **Step 4: Write room.component.html (new shell layout)**

```html
<div class="room-shell">
  <app-room-sidebar
    [isMicActive]="isMicActive"
    [isMuted]="isMuted"
    [isScreenSharing]="isScreenSharing"
    [isCameraActive]="isCameraActive"
    [isVoiceBusy]="isVoiceBusy"
    [isScreenShareBusy]="isScreenShareBusy"
    [isCameraBusy]="isCameraBusy"
    [avatarUrl]="currentUserAvatar"
    [displayName]="currentUserDisplayName"
    (back)="onBackToLobbyClick()"
    (voiceChannel)="onVoiceChannelClick()"
    (muteToggle)="onMuteToggleClick()"
    (screenShareToggle)="onScreenShareToggleClick()"
    (cameraToggle)="onCameraToggleClick()">
  </app-room-sidebar>

  <main class="room-main">
    <app-room-voice-pills></app-room-voice-pills>
    <app-room-stage>
      <!-- screenshare/video media slot: legacy voice-grid surfaces until parity -->
    </app-room-stage>
  </main>

  @if (isQueueOpen) {
    <aside class="room-queue-panel">
      <app-room-queue></app-room-queue>
    </aside>
  }

  <div class="room-player-bar">
    <app-room-player-bar></app-room-player-bar>
  </div>

  @if (isChatOpen) {
    <div class="room-chat-dock">
      <app-room-chat [isOpen]="isChatOpen"></app-room-chat>
    </div>
  }
</div>
```

- [ ] **Step 5: Write room.component.css (new shell)**

```css
:host { display: block; height: 100vh; height: 100dvh; width: 100%; overflow: hidden; }
.room-shell {
  display: grid;
  grid-template-columns: auto 1fr auto;
  grid-template-rows: 1fr auto;
  grid-template-areas:
    "sidebar main queue"
    "player  player player";
  height: 100vh; height: 100dvh;
  background: radial-gradient(circle at 75% 20%, rgba(45, 212, 191, 0.1) 0%, transparent 55%),
              radial-gradient(circle at 25% 80%, rgba(99, 102, 241, 0.06) 0%, transparent 45%),
              var(--bg-base);
}
app-room-sidebar { grid-area: sidebar; }
.room-main { grid-area: main; position: relative; min-width: 0; overflow: hidden; }
.room-queue-panel {
  grid-area: queue; width: 296px; flex-shrink: 0; overflow: hidden;
  background: var(--bg-glass); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
  border-left: 1px solid var(--border-color);
  animation: panel-slide-in var(--transition-panel);
}
.room-player-bar { grid-area: player; }
.room-chat-dock {
  position: absolute; top: 0; right: 0; bottom: 80px; width: 340px; z-index: var(--z-overlay);
  background: var(--bg-glass); backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
  border-left: 1px solid var(--border-color);
  animation: panel-slide-in var(--transition-panel);
}
@keyframes panel-slide-in { from { opacity: 0; transform: translateX(20px); } to { opacity: 1; transform: translateX(0); } }
@media (prefers-reduced-motion: reduce) { .room-queue-panel, .room-chat-dock { animation: none; } }

@media (max-width: 1120px) { .room-queue-panel { display: none; } }
@media (max-width: 768px) {
  .room-shell { grid-template-columns: auto 1fr; grid-template-areas: "sidebar main" "player player"; }
  .room-chat-dock { width: 100%; bottom: 0; }
}
```

- [ ] **Step 6: Update room.component.spec.ts**

Replace the existing spec with a minimal creation test for the new shell (the legacy spec content no longer applies). Ensure it imports the component and just asserts creation succeeds. (Avoid deep HTTP testing here — parity is verified manually in Task 19.)

```typescript
import { TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { RoomComponent } from './room.component';

describe('RoomComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [RouterTestingModule, RoomComponent]
  }));

  it('creates', () => {
    const fixture = TestBed.createComponent(RoomComponent);
    expect(fixture.componentInstance).toBeTruthy();
  });
});
```

- [ ] **Step 7: Verify build + run all room/shared tests**

Run: `npx ng build`
Expected: success.

Run: `npx ng test --watch=false --browsers=ChromeHeadless`
Expected: all green.

- [ ] **Step 8: Commit**

```bash
git add web-client/src/app/features/room/
git commit -m "refactor(room): rebuild Room shell (Cool Ocean player-centric), keep legacy as fallback"
```

---

## Task 19: Manual parity verification checklist

The engine, voice video-track attach, and live WebSockets cannot be meaningfully unit-tested in this environment. Verify manually against the running backend.

**Files:** none (verification only).

- [ ] **Step 1: Start the app**

Run: `npx ng serve` (from `web-client/`)
Open: `http://localhost:4200`

- [ ] **Step 2: Verify the parity checklist from the spec (section 8)**

Go through each item in the browser:
- [ ] Auth → Dashboard → create/enter a Room loads the new shell.
- [ ] **Voice**: join/leave channel works; mute/unmute; camera on/off; screen share toggle — each reflects in the sidebar active states.
- [ ] **Playback**: play/pause syncs across two browser tabs; seek; next/prev buttons present (no-op OK for now); volume slider controls all three player sources (try a YouTube, a SoundCloud, and a direct audio URL in the queue).
- [ ] **Chat**: messages send/receive realtime; when chat dock closed, unread badge increments; opening resets unread.
- [ ] **Queue**: add by URL and by search; vote; remove; drag-reorder (if `canManageQueue`); members tab + kick.
- [ ] **Layout state**: `music-only` → `music-voice` → `screenshare` → `video` transitions correctly when participants share/camera.
- [ ] **Responsive**: shrink to tablet (queue hides), mobile (chat dock full-width, volume hidden).
- [ ] **Reduced motion**: enable OS reduced-motion; visualizer/halo/ring animations stop, panels still appear.
- [ ] **A11y**: Tab through player bar/sidebar — teal focus rings visible; icon buttons announce labels.

- [ ] **Step 3: If any item fails, file as a follow-up task**

Do not delete legacy yet. Record failures and fix in a follow-up before Task 20.

---

## Task 20: Remove legacy fallback + temporary token alias

Only after Task 19 passes fully.

**Files:**
- Delete: `web-client/src/app/features/room/room-legacy.component.{ts,html,css,spec.ts}`
- Delete: `web-client/src/app/features/room/components/voice-grid/` (replaced by voice-pills; only remove after verifying screenshare/video media slot parity — if Stage media slot still depends on voice-grid video attach, keep voice-grid until that is wired, and drop this deletion into a follow-up).
- Modify: `web-client/src/design-tokens.css` — remove `--accent-alpha` legacy alias.

- [ ] **Step 1: Delete legacy component files**

```bash
git rm web-client/src/app/features/room/room-legacy.component.ts \
       web-client/src/app/features/room/room-legacy.component.html \
       web-client/src/app/features/room/room-legacy.component.css \
       web-client/src/app/features/room/room-legacy.component.spec.ts
```

- [ ] **Step 2: Remove the legacy alias from design-tokens.css**

Delete the line:
```css
  --accent-alpha:   rgba(45, 212, 191, 0.12);
```

- [ ] **Step 3: Confirm no references remain**

```bash
git grep -n "accent-alpha\|app-room-legacy\|RoomLegacyComponent" -- web-client/src
```
Expected: no output.

- [ ] **Step 4: Verify build**

Run: `npx ng build`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add -A web-client/src
git commit -m "chore(room): remove legacy Room fallback and accent-alpha alias"
```

---

## Notes for the implementing engineer

- **Player engine element IDs are load-bearing.** The hidden `youtube-player-element`, `soundcloud-player-element`, `html5-audio-element` must exist in the DOM (PlayerBar renders them) before `engine.initPlayers()` runs — that's why `initPlayers()` is deferred via `setTimeout` in `PlayerBarComponent.ngAfterViewInit`. Do not change these IDs.
- **`VoiceService.participants$`** items have fields `sid`, `identity`, `isMuted`, `isScreenSharing`, `isCameraOn`, `isLocal`, `raw`. The Stage/VoicePills rely on `isScreenSharing`/`isCameraOn` to pick the stage mode.
- **Do not touch core services.** If a service change seems necessary, stop and flag it — this plan's scope is `features/room` + `shared` + tokens only.
- **Violet hex audit:** after Task 17, the only allowed violet is in `--ocean-indigo` (intentional accent gradient end). Use `git grep -nE "6c63ff|7c74ff|rgba\(108, ?99, ?255"` to hunt stragglers.
- **Commit often.** Every task ends with a commit; do not batch.
