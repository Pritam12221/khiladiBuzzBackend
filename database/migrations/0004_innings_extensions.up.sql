BEGIN;


ALTER TABLE innings ADD COLUMN created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_innings_match_id ON innings(match_id);

COMMIT;
