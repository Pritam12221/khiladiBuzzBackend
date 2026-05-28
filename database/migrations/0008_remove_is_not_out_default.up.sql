BEGIN;

ALTER TABLE player_match_stats ALTER COLUMN is_not_out DROP DEFAULT;


UPDATE player_match_stats 
SET is_not_out = NULL 
WHERE dismissal_type IS NULL AND is_not_out = FALSE;

COMMIT;
