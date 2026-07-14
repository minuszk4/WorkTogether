# Realtime translation

Chat is delivered first. Translation runs in a bounded background queue and is dropped on timeout, saturation, or the configured character budget, so it cannot delay chat, music playback, or LiveKit media.

Each member selects a preferred language in Profile settings. Only that member's connected clients receive their translated result. Room membership is still required for every WebSocket event.

To enable Google Cloud Translation, grant the service account access to Cloud Translation, set `GOOGLE_SERVICE_ACCOUNT_FILE` in `.env`, then start the optional override:

```sh
docker compose -f docker-compose.yml -f docker-compose.translation.yml up -d
```

Do not commit the JSON key. Without the mount, the translation worker stays disabled and normal chat continues to work.

`subtitle:transcript` and `subtitle:received` are the text-only caption contract. They intentionally do not capture, store, or relay audio; an opt-in media/STT agent can submit transcripts without touching the music or playback path.
