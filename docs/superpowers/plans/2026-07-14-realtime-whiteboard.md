# Realtime Whiteboard Implementation Plan

**Goal:** Add a room-scoped collaborative canvas without changing music or playback paths.

**Architecture:** Reuse the authorised collab WebSocket for bounded drawing-operation events. Clients render locally and request a persisted snapshot on join.

## Tasks

- [ ] Add `whiteboard_snapshot` persistence and collab WebSocket `whiteboard.*` event validation.
- [ ] Add a standalone Angular canvas component that sends strokes and replays remote strokes.
- [ ] Mount the component in the room collaborative stage behind an explicit open/close control.
- [ ] Test operation validation, room membership, remote replay, and verify service/frontend builds.
