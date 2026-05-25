package dbhelper

import (
	"fmt"
	"strings"

	db "khiladiBuzz/database"
	"khiladiBuzz/models"
)


func CreateMatch(req models.CreateMatchRequest) (string, error) {

	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// get captains
	var team1CaptainID, team2CaptainID ,matchID,commonPlayerID string

	captainQuery := `SELECT captain_id FROM teams WHERE id = $1`

	if err = tx.Get(&team1CaptainID, captainQuery, req.Team1ID); err != nil {
		return "", fmt.Errorf("failed to get team1 captain: %w", err)
	}
	if err = tx.Get(&team2CaptainID, captainQuery, req.Team2ID); err != nil {
		return "", fmt.Errorf("failed to get team2 captain: %w", err)
	}

	//get common player

	fmt.Print("common pl",commonPlayerID)

	if req.CommonPlayerID != "" {
		commonPlayerID = req.CommonPlayerID
	}


	
//insert into match table
	matchQuery := `
		INSERT INTO matches (
			team1_id,
			team2_id,
			team1_captain_id,
			team2_captain_id,
			total_overs,
			toss_winner_team_id,
			toss_decision,
			status,
			common_player_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err = tx.Get(
		&matchID,
		matchQuery,
		req.Team1ID,
		req.Team2ID,
		team1CaptainID,
		team2CaptainID,
		req.TotalOvers,
		req.TossWinnerTeamID,
		req.TossDecision,
		req.Status,
		commonPlayerID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create match: %w", err)
	}

	
	type playerSlot struct {
		teamID   string
		playerID string
	}

	var slots []playerSlot
	for _, pid := range req.Team1PlayerIDs {
		slots = append(slots, playerSlot{req.Team1ID, pid})
	}
	for _, pid := range req.Team2PlayerIDs {
		slots = append(slots, playerSlot{req.Team2ID, pid})
	}

	if len(slots) > 0 {
		values := []string{}
		args := []interface{}{}
		argPos := 1

		for _, s := range slots {
			values = append(values, fmt.Sprintf("($%d, $%d, $%d)", argPos, argPos+1, argPos+2))
			args = append(args, matchID, s.teamID, s.playerID)
			argPos += 3
		}

		insertPlayersQuery := fmt.Sprintf(`
			INSERT INTO match_players (match_id, team_id, player_id)
			VALUES %s
			ON CONFLICT (match_id, team_id, player_id) DO NOTHING
		`, strings.Join(values, ", "))

		_, err = tx.Exec(insertPlayersQuery, args...)
		if err != nil {
			return "", fmt.Errorf("failed to insert match players: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return matchID, nil
}
