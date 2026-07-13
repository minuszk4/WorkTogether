CREATE TABLE IF NOT EXISTS room_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    created_by UUID NOT NULL,
    title VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    starts_at TIMESTAMPTZ NOT NULL,
    reminder_sent_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_room_events_upcoming ON room_events (room_id, starts_at) WHERE cancelled_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_room_events_reminders ON room_events (starts_at) WHERE cancelled_at IS NULL AND reminder_sent_at IS NULL;
