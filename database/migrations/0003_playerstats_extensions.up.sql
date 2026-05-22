BEGIN;


ALTER TABLE player_stats ALTER COLUMN user_id SET NOT NULL;


ALTER TABLE player_stats ADD COLUMN career_matches INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_runs INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_balls_faced INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_fours INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_sixes INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_wickets INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_balls_bowled INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_runs_given INT DEFAULT 0;
ALTER TABLE player_stats ADD COLUMN career_wins INT DEFAULT 0;

COMMIT;