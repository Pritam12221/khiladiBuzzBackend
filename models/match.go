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
	Striker    *PlayerStatsSummary `json:"striker,omitempty"`
	NonStriker *PlayerStatsSummary `json:"non_striker,omitempty"`
	Bowler     *PlayerStatsSummary `json:"bowler,omitempty"`
}

type CreateInningsRequest struct {
	InningsNumber int    `json:"innings_number" binding:"required"`
	BattingTeamID string `json:"batting_team_id" binding:"required"`
	BowlingTeamID string `json:"bowling_team_id" binding:"required"`
	Status        string `json:"status" binding:"required,oneof=live completed"`
}