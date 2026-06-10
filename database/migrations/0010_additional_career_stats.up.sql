BEGIN;

ALTER TABLE player_stats ADD COLUMN career_ducks INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_fifties INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_hundreds INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_highest_score INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_maidens INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_highest_wickets INT DEFAULT 0;

COMMIT;
