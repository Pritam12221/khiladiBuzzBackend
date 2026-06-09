package models

type CreateMatchRequest struct {
	Team1ID          string   `json:"team1_id" db:"team1_id" binding:"required"`
	Team2ID          string   `json:"team2_id" db:"team2_id" binding:"required"`
	TotalOvers       int      `json:"total_overs" db:"total_overs" binding:"required"`
	TossWinnerTeamID string   `json:"toss_winner_team_id" db:"toss_winner_team_id" binding:"required"`
	TossDecision     string   `json:"toss_decision" db:"toss_decision" binding:"required,oneof=bat bowl"`
	Status           string   `json:"status" db:"status" binding:"required,oneof=scheduled live completed cancelled"`
	Team1PlayerIDs   []string `json:"team1_player_ids" binding:"required"`
	Team2PlayerIDs   []string `json:"team2_player_ids" binding:"required"`
	CommonPlayerID   string   `json:"common_player_id"`
	StrikerID        string   `json:"striker_id"`
	NonStrikerID     string   `json:"non_striker_id"`
	BowlerID         string   `json:"bowler_id"`
}


type RecordBallRequest struct {
	OverNumber        int     `json:"over_number" binding:"min=0"`
	BallNumber        int     `json:"ball_number" binding:"required,min=1"`
	StrikerID         string  `json:"striker_id" binding:"required"`
	NonStrikerID      *string `json:"non_striker_id" binding:"omitempty"`
	BowlerID          string  `json:"bowler_id" binding:"required"`
	RunsScored        int     `json:"runs_scored"`
	ExtrasRuns        int     `json:"extras_runs"`
	ExtraType         *string `json:"extra_type"`          
	IsWicket          bool    `json:"is_wicket"`
	DismissalType     *string `json:"dismissal_type"`      
	DismissedPlayerID *string `json:"dismissed_player_id"` 
	FielderID         *string `json:"fielder_id" binding:"omitempty"`
}

type PlayerStatsSummary struct {
	PlayerID      string   `json:"player_id" db:"player_id"`
	RunsScored    int      `json:"runs_scored" db:"runs_scored"`
	BallsFaced    int      `json:"balls_faced" db:"balls_faced"`
	Fours         int      `json:"fours" db:"fours"`
	Sixes         int      `json:"sixes" db:"sixes"`
	IsNotOut      bool     `json:"is_not_out" db:"is_not_out"`
	DismissalType *string  `json:"dismissal_type" db:"dismissal_type"`
	DismissedBy   *string  `json:"dismissed_by" db:"dismissed_by"`
	RunsGiven     int      `json:"runs_given" db:"runs_given"`
	WicketsTaken  int      `json:"wickets_taken" db:"wickets_taken"`
	OversBowled   float64  `json:"overs_bowled" db:"overs_bowled"`
}

type RecordBallResponseDetails struct {
	Striker          *PlayerStatsSummary `json:"striker,omitempty"`
	NonStriker       *PlayerStatsSummary `json:"non_striker,omitempty"`
	Bowler           *PlayerStatsSummary `json:"bowler,omitempty"`
	NextStrikerID    *string             `json:"next_striker_id,omitempty"`
	NextNonStrikerID *string             `json:"next_non_striker_id,omitempty"`
	NextBowlerID     *string             `json:"next_bowler_id,omitempty"`
}

type CreateInningsRequest struct {
	InningsNumber int    `json:"innings_number" binding:"required"`
	BattingTeamID string `json:"batting_team_id" binding:"required"`
	BowlingTeamID string `json:"bowling_team_id" binding:"required"`
	Status        string `json:"status" binding:"required,oneof=live completed"`
	StrikerID     string `json:"striker_id"`
	NonStrikerID  string `json:"non_striker_id"`
	BowlerID      string `json:"bowler_id"`
}

type UpdateActivePlayersRequest struct {
	ActiveStrikerID    *string `json:"active_striker_id"`
	ActiveNonStrikerID *string `json:"active_non_striker_id"`
	ActiveBowlerID     *string `json:"active_bowler_id"`
}

type RetireHurtRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	MatchID  string `json:"match_id" binding:"required"`
}

type MatchListItem struct {
	ID               string   `json:"id" db:"id"`
	Team1Name        string   `json:"team1_name" db:"team1_name"`
	Team2Name        string   `json:"team2_name" db:"team2_name"`
	Team1ID          string   `json:"team1_id" db:"team1_id"`
	Team2ID          string   `json:"team2_id" db:"team2_id"`
	Status           string   `json:"status" db:"status"`
	TotalOvers       int      `json:"total_overs" db:"total_overs"`
	MatchDate        *string  `json:"match_date" db:"match_date"`
	TossWinnerTeamID *string  `json:"toss_winner_team_id" db:"toss_winner_team_id"`
	TossDecision     *string  `json:"toss_decision" db:"toss_decision"`
	WinnerTeamID     *string  `json:"winner_team_id" db:"winner_team_id"`
	Innings1Runs     *int     `json:"innings1_runs" db:"innings1_runs"`
	Innings1Wickets  *int     `json:"innings1_wickets" db:"innings1_wickets"`
	Innings1Overs    *float64 `json:"innings1_overs" db:"innings1_overs"`
	Innings2Runs     *int     `json:"innings2_runs" db:"innings2_runs"`
	Innings2Wickets  *int     `json:"innings2_wickets" db:"innings2_wickets"`
	Innings2Overs    *float64 `json:"innings2_overs" db:"innings2_overs"`
}

type InningsPlayersDetails struct {
	MatchID            string   `json:"match_id"`
	InningsNumber      int      `json:"innings_number"`
	MatchStatus        string   `json:"match_status"`
	InningsStatus      string   `json:"innings_status"`
	BattingTeamID      string   `json:"batting_team_id"`
	BowlingTeamID      string   `json:"bowling_team_id"`
	BattingTeamName    string   `json:"batting_team_name"`
	BowlingTeamName    string   `json:"bowling_team_name"`
	BattingPlayers     []MatchRosterPlayer `json:"batting_players"`
	BowlingPlayers     []MatchRosterPlayer `json:"bowling_players"`
	ActiveStrikerID    *string             `json:"active_striker_id"`
	ActiveNonStrikerID *string  `json:"active_non_striker_id"`
	ActiveBowlerID     *string  `json:"active_bowler_id"`
	StrikerStats       *PlayerStatsSummary `json:"striker_stats"`
	NonStrikerStats    *PlayerStatsSummary `json:"non_striker_stats"`
	TotalRuns          int      `json:"total_runs"`
	TotalWickets       int      `json:"total_wickets"`
	TotalOvers         float64  `json:"total_overs"`
	TotalOversLimit    int      `json:"total_overs_limit"`
	TossWinnerTeamID   *string  `json:"toss_winner_team_id"`
	TossDecision       *string  `json:"toss_decision"`
	TargetScore        *int     `json:"target_score"`
	FirstInningsRuns   *int     `json:"first_innings_runs"`
	FirstInningsWickets *int     `json:"first_innings_wickets"`
	BowlerStats        []BowlStat `json:"bowler_stats"`
}