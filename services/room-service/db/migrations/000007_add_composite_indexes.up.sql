CREATE INDEX IF NOT EXISTS idx_room_members_composite ON room_members(room_id, joined_at DESC);
