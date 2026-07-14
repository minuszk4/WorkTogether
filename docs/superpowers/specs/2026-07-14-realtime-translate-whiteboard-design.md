# Realtime translation and whiteboard

## Goal

Add a room whiteboard plus per-user translation for chat text and voice subtitles without affecting music playback latency.

## Architecture

Whiteboard operations use the existing authorised collaboration WebSocket as compact room events. The server persists a snapshot/checkpoint and broadcasts only drawing operations; clients render locally.

Translation runs as an independent worker path. Chat and voice services publish bounded text jobs containing room, source text, target language, and event ID. Each user receives translated output only for their saved preferred language. The original chat message or voice transcript always arrives first.

Voice audio, music audio, playback synchronisation, and translation never share a request path. Translation is best-effort: jobs exceeding the latency limit are dropped, not retried synchronously.

## User flow

- A user sets a preferred subtitle/chat language in settings.
- Chat shows original text immediately, followed by an optional translated line for that user.
- Voice shows the source transcript and an optional translated subtitle as separate realtime events.
- The room whiteboard can be opened by room members; all drawing edits appear live.

## Reliability and privacy

- Authenticate and verify room membership before whiteboard or translation subscriptions.
- Do not persist raw voice audio in the translation path; retain text only under existing chat/transcript retention rules.
- Limit translation concurrency per room and per user; cap event size and rate.
- Include original event IDs to deduplicate and preserve ordering client-side.

## Validation

- Unit tests for language preference, membership checks, rate limits, and dropped late jobs.
- Integration tests showing translated events cannot delay music/playback events.
- Browser tests for per-user language rendering and whiteboard operation replay.
