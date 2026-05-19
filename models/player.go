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
	Role        string `json:"role" `
}
