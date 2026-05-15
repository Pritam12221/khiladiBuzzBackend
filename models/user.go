package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" db:"phone_number" binding:"required"`
	Password    string `json:"password" db:"password" binding:"required"`
}
	
type User struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	PhoneNumber string `json:"phone_number" db:"phone_number"`
	Password    string `json:"password" db:"password"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at" db:"archived_at"`
}

type UserRequest struct {
	Name        string `json:"name" db:"name" binding:"required,min=3"`
	Password    string `json:"password" db:"password" binding:"required,min=6"`
	PhoneNumber string `json:"phone_number" db:"phone_number" binding:"required"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	SessionID string `json:"session_id"`

	jwt.RegisteredClaims
}


type CreateTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"`
	CaptainNumber string `json:"captain_number" binding:"required"`
	CaptainName   string `json:"captain_name"`
	Players []CreatePlayerRequest `json:"players" binding:"required"`
}

type Team struct {
	ID         string `db:"id" json:"id"`
	TeamName   string `db:"team_name" json:"team_name"`
	CaptainID  string `db:"captain_id" json:"captain_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Player struct {
	ID           string  `db:"id" json:"id"`
	PlayerName   string  `db:"player_name" json:"player_name"`
	PhoneNumber  *string `db:"phone_number" json:"phone_number"`
}

type CreatePlayerRequest struct {
	PlayerName  string `json:"player_name" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	Role        string `json:"role" `
}