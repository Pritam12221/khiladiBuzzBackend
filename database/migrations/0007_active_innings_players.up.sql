BEGIN;


ALTER TABLE innings 
ADD COLUMN IF NOT EXISTS active_striker_id UUID REFERENCES player_stats(id),
ADD COLUMN IF NOT EXISTS active_non_striker_id UUID REFERENCES player_stats(id),
ADD COLUMN IF NOT EXISTS active_bowler_id UUID REFERENCES player_stats(id);

COMMIT;
