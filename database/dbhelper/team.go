package dbhelper

import (
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
)

func GetPlayerByPhone(
	phone string,
) (models.Player, error) {

	var player models.Player

	query := `
		SELECT id, player_name, phone_number
		FROM players
		WHERE phone_number = $1
	`

	err := db.KhiladiDb.Get(
		&player,
		query,
		phone,
	)

	return player, err
}

func CreatePlayer(
	name string,
	phone string,
) (models.Player, error) {

	var player models.Player

	query := `
		INSERT INTO players (
			player_name,
			phone_number
		)
		VALUES ($1, $2)
		RETURNING id, player_name, phone_number
	`

	err := db.KhiladiDb.Get(
		&player,
		query,
		name,
		phone,
	)

	return player, err
}


func FindOrCreatePlayer(
	name string,
	phone string,
) (models.Player, error) {

	player, err := GetPlayerByPhone(phone)

	// player exists
	if err == nil {
		return player, nil
	}

	// create Player
	player, err = CreatePlayer(name, phone)

	return player, err
}


func CreateTeam(
	teamName string,
	captainID string,
	createdBy string,
) (string, error) {

	var teamID string

	query := `
		INSERT INTO teams (
			team_name,
			captain_id,
			created_by
		)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err := db.KhiladiDb.QueryRow(
		query,
		teamName,
		captainID,
		createdBy,
	).Scan(&teamID)

	return teamID, err
}


func AddPlayerToTeam(
	teamID string,
	playerID string,
) error {

	query := `
		INSERT INTO team_players (
			team_id,
			player_id
		)
		VALUES ($1, $2)
	`

	_, err := db.KhiladiDb.Exec(
		query,
		teamID,
		playerID,
	)

	return err
}