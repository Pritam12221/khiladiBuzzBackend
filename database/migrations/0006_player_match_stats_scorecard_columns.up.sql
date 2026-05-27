BEGIN;

ALTER TABLE player_match_stats
ADD COLUMN IF NOT EXISTS dismissal_type dismissal_type_enum;

ALTER TABLE player_match_stats
ADD COLUMN IF NOT EXISTS dismissed_by UUID REFERENCES player_stats(id);

ALTER TABLE player_match_stats
ADD COLUMN IF NOT EXISTS fielder_id UUID REFERENCES player_stats(id);

ALTER TABLE player_match_stats
ADD COLUMN IF NOT EXISTS is_not_out BOOLEAN DEFAULT FALSE;

ALTER TABLE player_match_stats
ADD COLUMN IF NOT EXISTS overs_bowled FLOAT DEFAULT 0;

COMMIT;
