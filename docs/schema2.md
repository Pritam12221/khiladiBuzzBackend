erDiagram

    %% =========================
    %% AUTHENTICATION
    %% =========================

    USERS ||--o{ USER_SESSION : "has"

    %% =========================
    %% USER / PLAYER RELATION
    %% =========================

    USERS ||--o| PLAYERS : "owns profile"

    %% =========================
    %% TEAM MANAGEMENT
    %% =========================

    TEAMS ||--o{ TEAM_PLAYERS : "contains"
    PLAYERS ||--o{ TEAM_PLAYERS : "joins"

    %% =========================
    %% MATCH MANAGEMENT
    %% =========================

    USERS ||--o{ MATCHES : "hosts"
    USERS ||--o{ MATCHES : "umpires"

    TEAMS ||--o{ MATCHES : "plays as team1"
    TEAMS ||--o{ MATCHES : "plays as team2"

    MATCHES ||--o{ MATCH_PLAYERS : "includes"

    PLAYERS ||--o{ MATCH_PLAYERS : "participates"

    %% =========================
    %% INNINGS & BALLS
    %% =========================

    MATCHES ||--|{ INNINGS : "contains"

    INNINGS ||--|{ BALLS : "contains"

    PLAYERS ||--o{ BALLS : "striker"
    PLAYERS ||--o{ BALLS : "non-striker"
    PLAYERS ||--o{ BALLS : "bowler"
    PLAYERS ||--o{ BALLS : "dismissed"

    %% =========================
    %% STATS & FANTASY
    %% =========================

    MATCHES ||--o{ PLAYER_MATCH_STATS : "creates"

    PLAYERS ||--o{ PLAYER_MATCH_STATS : "has"

    MATCHES ||--o{ PLAYER_MATCH_POINTS : "tracks"

    PLAYERS ||--o{ PLAYER_MATCH_POINTS : "earns"

    %% =========================
    %% ENTITIES
    %% =========================

    USERS {
        uuid id PK
        string name
        string phone_number UK
        string password
        datetime created_at
        datetime updated_at
    }

    USER_SESSION {
        uuid id PK
        uuid user_id FK
        datetime archived_at
        datetime created_at
    }

    PLAYERS {
        uuid id PK
        uuid user_id FK
        string player_name
        string phone_number
        string role
        datetime created_at
    }

    TEAMS {
        uuid id PK
        string team_name
        uuid captain_id FK
        uuid created_by FK
        datetime created_at
    }

    TEAM_PLAYERS {
        uuid id PK
        uuid team_id FK
        uuid player_id FK
        datetime joined_at
    }

    MATCHES {
        uuid id PK
        uuid team1_id FK
        uuid team2_id FK
        uuid host_id FK
        uuid umpire_id FK
        datetime match_date
        int total_overs
        uuid toss_winner_team_id FK
        string toss_decision
        uuid winner_team_id FK
        string status
        datetime created_at
    }

    MATCH_PLAYERS {
        uuid id PK
        uuid match_id FK
        uuid team_id FK
        uuid player_id FK
        boolean is_captain
        boolean is_playing
    }

    INNINGS {
        uuid id PK
        uuid match_id FK
        int innings_number
        uuid batting_team_id FK
        uuid bowling_team_id FK
        int total_runs
        int total_wickets
        float total_overs
        string status
    }

    BALLS {
        uuid id PK
        uuid innings_id FK

        int over_number
        int ball_number

        uuid striker_id FK
        uuid non_striker_id FK
        uuid bowler_id FK

        int runs_scored
        int extras_runs
        string extra_type

        boolean is_boundary
        int boundary_type

        boolean is_wicket
        string dismissal_type

        uuid dismissed_player_id FK

        datetime created_at
    }

    PLAYER_MATCH_STATS {
        uuid id PK

        uuid match_id FK
        uuid player_id FK

        int runs_scored
        int balls_faced
        int fours
        int sixes
        float strike_rate

        float overs_bowled
        int maiden_overs
        int runs_given
        int wickets_taken
        float economy
    }

    PLAYER_MATCH_POINTS {
        uuid id PK

        uuid player_id FK
        uuid match_id FK

        int batting_points
        int bowling_points
        int fielding_points
        int bonus_points
        int penalty_points

        int total_points

        datetime updated_at
    }