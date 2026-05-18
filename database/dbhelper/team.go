package dbhelper

import (
	"errors"
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"strings"

	"github.com/jmoiron/sqlx"
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

func CreatePlayerForUser(
	name string,
	phone string,
	userID string,
) (models.Player, error) {

	var player models.Player

	query := `
		INSERT INTO players (
			player_name,
			phone_number,
			user_id
		)
		VALUES ($1, $2, $3)
		RETURNING id, player_name, phone_number
	`

	err := db.KhiladiDb.Get(
		&player,
		query,
		name,
		phone,
		userID,
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
	captainId string,
	createdBy string,
) (string, error) {
	fmt.Println("user",createdBy);
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
		captainId,
		createdBy,
	).Scan(&teamID)
		fmt.Print(err);
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



func CreateTeamWithPlayers(
	req models.CreateTeamRequest,
	userID string,
) (string, error) {

	// collect phone numbers
	phones := []string{}

	for _, p := range req.Players {
		phones = append(phones, p.PhoneNumber)
	}

	// fetch existing players [slice of player]
	query, args, err := sqlx.In(`
		SELECT id, player_name, phone_number
		FROM players
		WHERE phone_number IN (?)
	`, phones)

	if err != nil {
		return "", err
	}

	query = db.KhiladiDb.Rebind(query)

	var existingPlayers []models.Player

	err = db.KhiladiDb.Select(
		&existingPlayers,
		query,
		args...,
	)

	if err != nil {
		return "", err
	}

	// phone -> player info
	playerMap := map[string]models.Player{}

	for _, p := range existingPlayers {
		playerMap[*p.PhoneNumber] = p
	}

	// identify missing players
	missingPlayers := []models.CreatePlayerRequest{}

	for _, p := range req.Players {

		_, exists := playerMap[p.PhoneNumber]

		if !exists {
			missingPlayers = append(
				missingPlayers,
				p,
			)
		}
	}

	// bulk insert missing players
	if len(missingPlayers) > 0 {

		values := []string{}
		insertArgs := []interface{}{}

		argPos := 1

		for _, p := range missingPlayers {

			values = append(
				values,
				fmt.Sprintf(
					"($%d, $%d, $%d)",
					argPos,
					argPos+1,
					argPos+2,
				),
			)

			insertArgs = append(
				insertArgs,
				p.PlayerName,
				p.PhoneNumber,
				p.Role,
			)

			argPos += 3
		}

		insertQuery := fmt.Sprintf(`
			INSERT INTO players (
				player_name,
				phone_number,
				role
			)
			VALUES %s
			RETURNING id, player_name, phone_number
		`, strings.Join(values, ","))

		var newPlayers []models.Player

		err = db.KhiladiDb.Select(
			&newPlayers,
			insertQuery,
			insertArgs...,
		)

		if err != nil {
			return "", err
		}

		// merge newly created players
		for _, p := range newPlayers {
			playerMap[*p.PhoneNumber] = p
		}
	}

	// determine captain id
	var captainID string

	for _, p := range req.Players {

		if p.PhoneNumber == req.CaptainNumber {

			player := playerMap[p.PhoneNumber]

			captainID = player.ID
			break
		}
	}

	// captain validation
	if captainID == "" {
		return "", errors.New(
			"captain must be part of players list",
		)
	}

	// create team
	teamID, err := CreateTeam(
		req.TeamName,
		captainID,
		userID,
	)

	if err != nil {
		return "", err
	}

	// add players to team_players
	for _, p := range req.Players {

		player := playerMap[p.PhoneNumber]

		err = AddPlayerToTeam(
			teamID,
			player.ID,
		)

		if err != nil {
			return "", err
		}
	}

	return teamID, nil
}
