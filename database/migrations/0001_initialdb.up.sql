
BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    phone_number VARCHAR(15) UNIQUE NOT NULL,
    password TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    archived_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);



-- PLAYERS
CREATE TYPE player_role_enum AS ENUM (
    'batsman',
    'bowler',
    'allrounder',
    'wicketkeeper'
);

CREATE TYPE batting_style_enum AS ENUM (
    'right_hand',
    'left_hand'
);



CREATE TYPE bowling_style_enum AS ENUM (
    'right_arm_pace',
    'left_arm_pace',
    'right_arm_spin',
    'left_arm_spin'
);

CREATE TABLE IF NOT EXISTS players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    player_name TEXT NOT NULL,
    phone_number VARCHAR(15),
    role player_role_enum DEFAULT 'allrounder',
    batting_style batting_style_enum ,
    bowling_style bowling_style_enum,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- TEAMS

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_name TEXT NOT NULL,
    captain_id UUID REFERENCES players(id),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


-- TEAM PLAYERS

CREATE TABLE IF NOT EXISTS team_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id),
    player_id UUID NOT NULL REFERENCES players(id),
    UNIQUE(team_id, player_id)
);


-- MATCHES

CREATE TYPE match_status_enum AS ENUM (
    'scheduled',
    'live',
    'completed',
    'cancelled'
);

CREATE TYPE toss_decision_enum AS ENUM (
    'bat',
    'bowl'
);

CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team1_id UUID NOT NULL REFERENCES teams(id),
    team2_id UUID NOT NULL REFERENCES teams(id),

    host_id UUID REFERENCES users(id),
    umpire_id UUID REFERENCES users(id),

    match_date TIMESTAMP WITH TIME ZONE,
    total_overs INT NOT NULL,
    toss_winner_team_id UUID REFERENCES teams(id),
    toss_decision TEXT,
    winner_team_id UUID REFERENCES teams(id),
    status match_status_enum DEFAULT 'scheduled',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_matches_status
ON matches(status);




-- INNINGS
CREATE TYPE innings_status_enum AS ENUM (
    'live',
    'completed'
);

CREATE TABLE IF NOT EXISTS innings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id),
    innings_number INT NOT NULL,
    batting_team_id UUID NOT NULL REFERENCES teams(id),
    bowling_team_id UUID NOT NULL REFERENCES teams(id),
    total_runs INT DEFAULT 0,
    total_wickets INT DEFAULT 0,
    total_overs FLOAT DEFAULT 0,
    status  innings_status_enum DEFAULT 'live',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- BALLS

CREATE TYPE extra_type_enum AS ENUM (
    'wide',
    'no_ball'
);


CREATE TYPE dismissal_type_enum AS ENUM (
    'bowled',
    'caught',
    'lbw',
    'runout',
    'stumped',
    'hit_wicket',
    'retired_hurt'
);

CREATE TABLE IF NOT EXISTS balls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    innings_id UUID NOT NULL REFERENCES innings(id),

    over_number INT NOT NULL,
    ball_number INT NOT NULL,

    striker_id UUID REFERENCES players(id),
    non_striker_id UUID REFERENCES players(id),
    bowler_id UUID REFERENCES players(id),

    runs_scored INT DEFAULT 0,
    extras_runs INT DEFAULT 0,

    extra_type extra_type_enum,
    is_wicket BOOLEAN DEFAULT FALSE,

    dismissal_type dismissal_type_enum,
    dismissed_player_id UUID REFERENCES players(id),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_balls_innings
ON balls(innings_id);

-- PLAYER MATCH STATS

CREATE TABLE IF NOT EXISTS player_match_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id),
    player_id UUID NOT NULL REFERENCES players(id),

    runs_scored INT DEFAULT 0,
    balls_faced INT DEFAULT 0,
    fours INT DEFAULT 0,
    sixes INT DEFAULT 0,

    maiden_overs INT DEFAULT 0,
    runs_given INT DEFAULT 0,
    wickets_taken INT DEFAULT 0,

    UNIQUE(match_id, player_id),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- PLAYER MATCH POINTS

CREATE TABLE IF NOT EXISTS player_match_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id),
    match_id UUID NOT NULL REFERENCES matches(id),

    batting_points INT DEFAULT 0,
    bowling_points INT DEFAULT 0,

    total_points INT DEFAULT 0,

    UNIQUE(match_id, player_id),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);



COMMIT;