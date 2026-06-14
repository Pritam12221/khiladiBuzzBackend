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
		SELECT p.id, u.name AS player_name, u.phone_number, p.user_id, p.role, p.batting_style, p.bowling_style, p.created_at, p.updated_at
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		WHERE u.phone_number = $1
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
	captainId string,
	createdBy string,
) (string, error) {
	fmt.Println("user", createdBy)
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
	fmt.Print(err)
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



	phones := []string{}

	for _, p := range req.Players {

		phones = append(
			phones,
			p.PhoneNumber,
		)
	}


	query, args, err := sqlx.In(`
		SELECT
			p.id,
			u.name AS player_name,
			u.phone_number,
			p.user_id
		FROM player_stats p
		JOIN users u
			ON p.user_id = u.id
		WHERE u.phone_number IN (?)
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


	playerMap := map[string]models.Player{}

	for _, p := range existingPlayers {

		if p.PhoneNumber != nil {

			playerMap[*p.PhoneNumber] = p
		}
	}

	// Validate that any existing phone no.
	for _, p := range req.Players {
		existing, exists := playerMap[p.PhoneNumber]
		if exists {
			if !strings.EqualFold(p.PlayerName, existing.PlayerName) {
				return "", fmt.Errorf("phone number %s is already registered to player %s", p.PhoneNumber, existing.PlayerName)
			}
		}
	}


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


	for _, p := range missingPlayers {

		newPlayer, err := CreateUser(
			p.PlayerName,
			p.PhoneNumber,
			"",
		)

		if err != nil {
			return "", err
		}


		if newPlayer.PhoneNumber != nil {
			playerMap[*newPlayer.PhoneNumber] = newPlayer
		}
	}



	var captainID string

	for _, p := range req.Players {

		if p.PhoneNumber == req.CaptainNumber {

			player := playerMap[p.PhoneNumber]

			captainID = player.ID

			break
		}
	}

	if captainID == "" {

		return "", errors.New(
			"invalid captain",
		)
	}



	teamID, err := CreateTeam(
		req.TeamName,
		captainID,
		userID,
	)

	if err != nil {
		return "", err
	}



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


func FetchTeams(userID string) ([]models.Team, error) {
	teams := []models.Team{}
	query := `
		SELECT id, team_name
		FROM teams
		WHERE created_by = $1
		ORDER BY created_at DESC
	`
	err := db.KhiladiDb.Select(&teams, query, userID)
	return teams, err
}

func FetchTeamPlayers(teamID string) ([]models.Player, error) {
	players := []models.Player{}
	query := `
		SELECT
			p.id,
			u.name  AS player_name,
			u.phone_number,
			p.user_id,
			p.role,
			p.batting_style,
			p.bowling_style
		FROM team_players tp
		JOIN player_stats p ON tp.player_id = p.id
		JOIN users u ON p.user_id = u.id
		WHERE tp.team_id = $1
		ORDER BY u.name ASC
	`
	err := db.KhiladiDb.Select(&players, query, teamID)
	return players, err
}

func FindOrCreatePlayerForTeam(
	name string,
	phone string,
	teamID string,
) (models.Player, error) {
	var player models.Player

	// Search player_stats + users by phone
	query := `
		SELECT p.id, u.name AS player_name, u.phone_number, p.user_id, p.role
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		WHERE u.phone_number = $1
	`
	err := db.KhiladiDb.Get(&player, query, phone)

	if err == nil {
		// Phone number exists, check if name matches
		if !strings.EqualFold(name, player.PlayerName) {
			return models.Player{}, fmt.Errorf("phone number %s is already registered to player %s", phone, player.PlayerName)
		}
	} else {
		player, err = CreateUser(name, phone, "")
		if err != nil {
			return models.Player{}, err
		}
	}



	// Link them to the team in team_players
	var exists bool
	checkQuery := `SELECT COUNT(*) > 0 FROM team_players WHERE team_id = $1 AND player_id = $2`
	err = db.KhiladiDb.Get(&exists, checkQuery, teamID, player.ID)
	if err == nil && !exists {
		err = AddPlayerToTeam(teamID, player.ID)
		if err != nil {
			return models.Player{}, fmt.Errorf("failed to link player to team: %w", err)
		}
	}

	return player, nil
}