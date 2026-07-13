ALTER TABLE rooms ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'chill';
ALTER TABLE rooms ADD CONSTRAINT rooms_mode_check CHECK (mode IN ('chill', 'focus', 'collaborate'));
