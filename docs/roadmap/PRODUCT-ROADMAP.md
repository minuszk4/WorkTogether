# WorkTogether — Product Roadmap

**Tạo:** 2026-06-20
**Trạng thái:** Living document — cập nhật khi mỗi pha hoàn thành
**Mục đích:** Biến WorkTogether từ "Spotify + Discord chạy song song" thành một sản phẩm có bản sắc riêng — nơi âm nhạc và sự cộng tác **cảm nhận được nhau**, không chỉ chạy cùng lúc.

---

## Triết lý sản phẩm

**WorkTogether ≠ Spotify + Discord.** Bản sắc nằm ở chỗ: sự hiện diện của người khác **thay đổi** trải nghiệm nghe, không chỉ là backdrop. Khi bạn vào phòng, bạn phải *cảm nhận được* phòng đang sống — ai đang nghe, ai đang cảm, phòng đang ở mood nào.

4 trụ cột tính năng phân biệt, theo thứ tự ưu tiên thực hiện:

| # | Nhóm | Đặc trưng | Backend |
|---|------|-----------|---------|
| A | **Presence & Vibe** | Phòng "sống", reaction realtime, listener state | Nhẹ (mở rộng WS) |
| B | **Nghe cùng nhau** | Synced lyrics, bookmark khoảnh khắc, DJ mode | Vừa |
| C | **Work/Study** | Pomodoro dùng chung, note collab, focus mode | Nặng |
| D | **Gắn kết & Ký ức** | Room stats, shared history, scheduling | Vừa |

---

## Roadmap 4 pha (tuần tự)

Mỗi pha là một vòng lặp hoàn chỉnh: **brainstorm → spec → plan → implement → verify**. Không bắt đầu pha sau trước khi pha trước đạt done.

### Pha 0 — Redesign FE (ĐANG DIỄN RA) ✏️
- **Mục tiêu:** Cool Ocean design system + layout player-centric cho Room.
- **Spec:** `docs/superpowers/specs/2026-06-20-room-redesign-cool-ocean-design.md`
- **Plan:** `docs/superpowers/plans/2026-06-20-room-redesign-cool-ocean.md` (20 tasks)
- **Lý do làm đầu tiên:** Tất cả pha sau build trên layout mới (avatar pills, visualizer, stage). Xây feature trên nền cũ = refactor lại sau.
- **Done khi:** Room mới đạt parity, legacy xóa, build xanh.

---

### Pha 1 — Presence & Vibe 🎯
*Làm phòng "sống". Đây là thứ Spotify không có, Discord thì yếu.*

- **Listener indicators** — mỗi thành viên hiện trạng thái nghe (playing/paused/seeking). Biết ai "đu" theo, ai lag.
- **Live reactions** — reaction (❤️🔥👏) bay qua player khi đoạn drop nổ (như IG Live/Twitch).
- **Speaker/reaction pulse** — avatar phóng to khi ai đó reaction hoặc đang nói.
- **Room vibe meter** — tâm trạng phòng realtime (chill/hype/study) aggregate từ reactions.

**Tại sao làm sau redesign:** Reactions + listener pills nằm sẵn trong layout mới (voice pills, visualizer, stage). Chi phí thấp, hiệu ứng thay đổi cảm giác căn phòng ngay lập tức.

**Backend cần:** Mở rộng `chat-ws` (hoặc `presence-ws` mới) thêm event: `listener:state`, `reaction:send`, `vibe:tick`. Không cần service mới.

---

### Pha 2 — Nghe Cùng Nhau 🎶
*Tăng chiều sâu trải nghiệm âm nhạc — không chỉ cùng lúc mà cùng cảm.*

- **Synced lyrics** — lời bài hát trượt theo playback (như Apple Music). Cùng hát/đọc.
- **Bookmark khoảnh khắc** — mark timestamp + note ("đoạn này hay"), share vào chat, jump tới được.
- **Up-next realtime poll** — mini-poll trước khi bài kết thúc: "Bài tiếp: A hay B?".
- **DJ mode / takeover** — host nhường quyền control playback cho thành viên vài phút (guest DJ).

**Tại sao làm thứ 2:** Tận dụng playback-ws + NTP sync đã có. Lyrics + bookmark là tính năng "wow" nhưng scope vừa phải.

**Backend cần:**
- Lyrics: API lấy lyrics theo track (third-party hoặc store), hoặc client-side fetch.
- Bookmark: bảng `room_bookmarks` (room_id, user_id, track_id, position_ms, note).
- DJ mode: mở rộng permission `CAN_CONTROL_PLAYBACK` thành token tạm thời (guest DJ token có TTL).

---

### Pha 3 — Work/Study 📚
*Lợi thế cạnh tranh dài hạn. Thứ khiến WorkTogether khác hẳn Spotify.*

- **Pomodoro dùng chung** — đồng hồ 25/5 synced cả phòng, nhạc tự fade khi break.
- **Bảng note/checklist collab** — sticky note realtime trong phòng (học nhóm/brainstorm).
- **Focus music mode** — playlist lofi/ambient + timer + "không skip" để ai cũng tập trung.
- **Co-queue fairness** — đảm bảo mỗi người được phát ≥1 bài, chống spam chiếm queue.

**Tại sao làm thứ 3:** Phức tạp nhất, cần subsystem riêng (collab doc realtime kiểu Google Docs + timer sync service). Lợi thế lớn nhưng rủi ro cao — nên làm khi nền tảng (Pha 0-2) đã vững.

**Backend cần (nặng):**
- `timer-service` mới (Go microservice): sync pomodoro state qua WS.
- `collab-service` mới: CRDT/OT cho note collab realtime (hoặc dùng lib như Yjs).
- Mở rộng permission: `CAN_START_POMODORO`, `CAN_EDIT_NOTES`.

---

### Pha 4 — Gắn Kết & Ký Ức 💜
*Retention — làm người dùng muốn quay lại.*

- **Room stats** — "Cả phòng nghe cùng nhau 47h tuần này", bài top.
- **Shared listening history** — timeline "chúng ta đã nghe gì cùng nhau".
- **Room scheduling** — lên lịch listening party / study session, notify khi đến giờ.
- **Room identity** — avatar phòng, mood/theme, rules hiển thị.

**Tại sao làm cuối:** Phụ thuộc data tích lũy từ Pha 1-3 (stats cần có reactions, history cần nhiều session). Làm sớm thì không có data để show.

**Backend cần:**
- `stats-service`: aggregation job chạy định kỳ.
- `scheduling-service` (hoặc mở rộng notification-service): cron + reminder.
- Mở rộng `rooms` table: avatar, theme, rules.

---

## Nguyên tắc thực hiện

1. **Tuần tự, không song song** — mỗi pha hoàn chỉnh trước khi qua kế. Tránh nửa vời tất cả.
2. **Mỗi pha = vòng brainstorm → spec → plan → implement → verify** — dùng skill `brainstorming` để chốt scope, rồi `writing-plans`.
3. **Backend song song FE khi cần** — nếu pha cần service mới, brainstorm cả 2 bên rồi chia task.
4. **YAGNI trong từng pha** — không nhồi mọi ý tưởng vào 1 pha. Cắt tinh giản trước khi spec.
5. **Cập nhật document này** sau mỗi pha (đánh dấu done, ghi lesson learned).

---

## Trạng thái pha

| Pha | Tên | Trạng thái | Spec | Plan |
|-----|-----|-----------|------|------|
| 0 | Redesign FE | 🔄 Đang làm | [link](../superpowers/specs/2026-06-20-room-redesign-cool-ocean-design.md) | [link](../superpowers/plans/2026-06-20-room-redesign-cool-ocean.md) |
| 1 | Presence & Vibe | ⏳ Chờ | — | — |
| 2 | Nghe Cùng Nhau | ⏳ Chờ | — | — |
| 3 | Work/Study | ⏳ Chờ | — | — |
| 4 | Gắn Kết & Ký Ức | ⏳ Chờ | — | — |

---

## Cách dùng document này

- Khi bắt đầu 1 pha: đọc lại triết lý + nguyên tắc, rồi invoke `brainstorming` skill để chốt scope chi tiết cho pha đó.
- Khi xong 1 pha: cập nhật "Trạng thái pha" (✅ done), ghi link spec/plan, thêm 1-2 dòng lesson learned.
- Document này là **north star** — khi lan man, quay lại đây để nhớ vì sao chọn thứ tự này.
