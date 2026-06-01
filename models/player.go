package models

import "time"

type Player struct {
	ID           string     `db:"id" json:"id"`
	UserID       *string    `db:"user_id" json:"user_id"`
	PlayerName   string     `db:"player_name" json:"player_name"`
	PhoneNumber  *string    `db:"phone_number" json:"phone_number"`
	Role         *string    `db:"role" json:"role"`
	BattingStyle *string    `db:"batting_style" json:"batting_style"`
	BowlingStyle *string    `db:"bowling_style" json:"bowling_style"`
	CreatedAt    *time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at" json:"updated_at"`

	// Career Aggregates
	CareerMatches     int `db:"career_matches" json:"career_matches"`
	CareerRuns        int `db:"career_runs" json:"career_runs"`
	CareerBallsFaced  int `db:"career_balls_faced" json:"career_balls_faced"`
	CareerFours       int `db:"career_fours" json:"career_fours"`
	CareerSixes       int `db:"career_sixes" json:"career_sixes"`
	CareerWickets     int `db:"career_wickets" json:"career_wickets"`
	CareerBallsBowled int `db:"career_balls_bowled" json:"career_balls_bowled"`
	CareerRunsGiven   int `db:"career_runs_given" json:"career_runs_given"`
	CareerWins        int `db:"career_wins" json:"career_wins"`
}

type UpdateProfileRequest struct {
	PlayerName   string  `json:"player_name" binding:"required"`
	PhoneNumber  string  `json:"phone_number" binding:"required"`
	Role         string  `json:"role" binding:"required,oneof=batsman bowler allrounder wicketkeeper"`
	BattingStyle *string `json:"batting_style" binding:"omitempty,oneof=right_hand left_hand"`
	BowlingStyle *string `json:"bowling_style" binding:"omitempty,oneof=right_arm_pace left_arm_pace right_arm_spin left_arm_spin"`
}

type CreatePlayerRequest struct {
	PlayerName  string `json:"player_name" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	Role        string `json:"role"`
}

type MatchRosterPlayer struct {
	ID           string  `db:"id" json:"id"`
	UserID       *string `db:"user_id" json:"user_id"`
	PlayerName   string  `db:"player_name" json:"player_name"`
	PhoneNumber  *string `db:"phone_number" json:"phone_number"`
	Role         *string `db:"role" json:"role"`
	BattingStyle *string `db:"batting_style" json:"batting_style"`
	BowlingStyle *string `db:"bowling_style" json:"bowling_style"`
}
