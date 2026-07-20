# Realtime translation

Chat is delivered first. Translation runs in a bounded background queue and is dropped on timeout, saturation, or the configured character budget, so it cannot delay chat, music playback, or LiveKit media.

Each member selects a preferred language in Profile settings. Only that member's connected clients receive their translated result. Room membership is still required for every WebSocket event.

To enable translation and voice captions, enable Cloud Translation and Speech-to-Text for the service account, set `GOOGLE_SERVICE_ACCOUNT_FILE` in `.env`, then start the optional override:

```sh
docker compose -f docker-compose.yml -f docker-compose.translation.yml up -d
```

Do not commit the JSON key. Without the mount, the translation worker stays disabled and normal chat continues to work.

Voice captions are opt-in. The browser reuses the microphone track already published to LiveKit, uploads a short WebM/Opus chunk, and discards it after Google Speech-to-Text returns. The server accepts at most 128 KB per chunk, limits each member to one chunk every two seconds, and processes at most two chunks at once. Audio is not stored or relayed, and music/playback is not in this path.
