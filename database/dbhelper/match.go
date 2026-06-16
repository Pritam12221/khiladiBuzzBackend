package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"strings"

	"github.com/jmoiron/sqlx"
)

// match creation inc innings ,fetch match captains,add active player to the match
func CreateMatch(req models.CreateMatchRequest, userID string) (string, string, error) {

	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	//  get captains
	var team1CaptainID, team2CaptainID, matchID string
	var commonPlayerID *string

	captainQuery := `SELECT captain_id FROM teams WHERE id = $1`

	if err = tx.Get(&team1CaptainID, captainQuery, req.Team1ID); err != nil {
		return "", "", fmt.Errorf("failed to get team1 captain: %w", err)
	}
	if err = tx.Get(&team2CaptainID, captainQuery, req.Team2ID); err != nil {
		return "", "", fmt.Errorf("failed to get team2 captain: %w", err)
	}

	if req.CommonPlayerID != "" {
		commonPlayerID = &req.CommonPlayerID
	}

	// insert match
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
			common_player_id,
			host_id,
			umpire_id,
			team1_size,
			team2_size
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
		userID,
		userID,
		req.Team1Size,
		req.Team2Size,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create match: %w", err)
	}

	// insert match players using extracted helper
	err = AddMatchPlayersTx(tx, matchID, req.Team1ID, req.Team1PlayerIDs, req.Team2ID, req.Team2PlayerIDs)
	if err != nil {
		return "", "", err
	}

	//toss decide
	var battingTeamID, bowlingTeamID string

	if req.TossDecision == "bat" {
		battingTeamID = req.TossWinnerTeamID
		if req.TossWinnerTeamID == req.Team1ID {
			bowlingTeamID = req.Team2ID
		} else {
			bowlingTeamID = req.Team1ID
		}
	} else {
		if req.TossWinnerTeamID == req.Team1ID {
			battingTeamID = req.Team2ID
			bowlingTeamID = req.Team1ID
		} else {
			battingTeamID = req.Team1ID
			bowlingTeamID = req.Team2ID
		}
	}

	// create innings using extracted helper
	inningsID, err := CreateInningsTx(tx, matchID, 1, battingTeamID, bowlingTeamID, "live", req.StrikerID, req.NonStrikerID, req.BowlerID)
	if err != nil {
		return "", "", fmt.Errorf("failed to create innings: %w", err)
	}

	if req.StrikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, req.StrikerID)
	}
	if req.NonStrikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, req.NonStrikerID)
	}
	if req.BowlerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, req.BowlerID)
	}

	if err = tx.Commit(); err != nil {
		return "", "", err
	}

	return matchID, inningsID, nil
}

func AddMatchPlayersTx(tx *sqlx.Tx, matchID string, team1ID string, team1PlayerIDs []string, team2ID string, team2PlayerIDs []string) error {
	type playerSlot struct {
		teamID   string
		playerID string
	}

	var slots []playerSlot
	for _, pid := range team1PlayerIDs {
		slots = append(slots, playerSlot{team1ID, pid})
	}
	for _, pid := range team2PlayerIDs {
		slots = append(slots, playerSlot{team2ID, pid})
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

		_, err := tx.Exec(insertPlayersQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to insert match players: %w", err)
		}
	}

	return nil
}

func FetchAllMatches(status string, limit, offset int) ([]models.MatchListItem, error) {
	matches := []models.MatchListItem{}
	query := `
		SELECT 
			m.id, 
			t1.team_name as team1_name, 
			t2.team_name as team2_name,
			m.team1_id,
			m.team2_id,
			m.status, 
			m.total_overs, 
			m.match_date,
			m.toss_winner_team_id, 
			m.toss_decision,
			m.winner_team_id,
			m.host_id,
			i1.total_runs as innings1_runs,
			i1.total_wickets as innings1_wickets,
			i1.total_overs as innings1_overs,
			i2.total_runs as innings2_runs,
			i2.total_wickets as innings2_wickets,
			i2.total_overs as innings2_overs
		FROM matches m
		JOIN teams t1 ON m.team1_id = t1.id
		JOIN teams t2 ON m.team2_id = t2.id
		LEFT JOIN innings i1 ON m.id = i1.match_id AND i1.innings_number = 1
		LEFT JOIN innings i2 ON m.id = i2.match_id AND i2.innings_number = 2
		WHERE ($1 = '' OR m.status::text = $1)
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := db.KhiladiDb.Select(&matches, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all matches: %w", err)
	}
	return matches, nil
}
