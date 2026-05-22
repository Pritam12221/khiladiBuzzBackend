BEGIN;


CREATE TABLE IF NOT EXISTS match_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    player_id UUID NOT NULL REFERENCES player_stats(id),
    is_captain BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(match_id, team_id, player_id)
);


ALTER TABLE matches
ADD COLUMN team1_captain_id UUID REFERENCES player_stats(id),
ADD COLUMN team2_captain_id UUID REFERENCES player_stats(id);


ALTER TABLE matches
ADD COLUMN best_player_id UUID REFERENCES player_stats(id),
ADD COLUMN worst_player_id UUID REFERENCES player_stats(id),
ADD COLUMN common_player_id UUID REFERENCES player_stats(id);

COMMIT;
