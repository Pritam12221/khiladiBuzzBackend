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
