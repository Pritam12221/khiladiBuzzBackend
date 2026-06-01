package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"
	"strings"

	"github.com/jmoiron/sqlx"
)


func UpdateInningsStatsTx(tx *sqlx.Tx, inningsID string, runsInc, wicketsInc int, isLegal bool) error {
	// Fetch current overs count
	var currentOvers float64
	err := tx.Get(&currentOvers, `SELECT total_overs FROM innings WHERE id = $1`, inningsID)
	if err != nil {
		return err
	}


	var newOvers float64
	if isLegal {
		completedOvers := int(currentOvers)
		balls := int(math.Round((currentOvers - float64(completedOvers)) * 10))

		totalBalls := completedOvers*6 + balls + 1
		newOvers = float64(totalBalls/6) + float64(totalBalls%6)*0.1
	} else {
		newOvers = currentOvers
	}

	_, err = tx.Exec(`
		UPDATE innings SET
			total_runs    = total_runs + $1,
			total_wickets = total_wickets + $2,
			total_overs   = $3,
			updated_at    = NOW()
		WHERE id = $4`,
		runsInc, wicketsInc, newOvers, inningsID,
	)
	return err
}

// UpdateNonStrikerStatsTx ensures the non-striker player has a player_match_stats row and is marked as not out.
func UpdateNonStrikerStatsTx(tx *sqlx.Tx, matchID string, nonStrikerID string) error {
	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, player_id)
VALUES ($1, $2)
ON CONFLICT (match_id, player_id) DO NOTHING`,
		matchID, nonStrikerID,
	)
	return err
}

// UpdateStrikerStatsTx updates the active striker's batting statistics in the player_match_stats table.
func UpdateStrikerStatsTx(tx *sqlx.Tx, matchID string, strikerID string, runsScored int, isBallFaced, isBye, isLegBye bool) error {
	runsForBatsman := runsScored
	if isBye || isLegBye {
		runsForBatsman = 0
	}

	foursInc, sixesInc := 0, 0
	if runsForBatsman == 4 {
		foursInc = 1
	} else if runsForBatsman == 6 {
		sixesInc = 1
	}

	ballsFacedInc := 0
	if isBallFaced {
		ballsFacedInc = 1
	}

	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, player_id, runs_scored, balls_faced, fours, sixes)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (match_id, player_id) DO UPDATE SET
			runs_scored = player_match_stats.runs_scored + EXCLUDED.runs_scored,
			balls_faced = player_match_stats.balls_faced + EXCLUDED.balls_faced,
			fours       = player_match_stats.fours       + EXCLUDED.fours,
			sixes       = player_match_stats.sixes       + EXCLUDED.sixes,
			updated_at  = NOW()`,
		matchID, strikerID, runsForBatsman, ballsFacedInc, foursInc, sixesInc,
	)
	return err
}

// UpdateBowlerStats
func UpdateBowlerStatsTx(tx *sqlx.Tx, matchID string, req models.RecordBallRequest, isLegal, isWide, isNoBall, isBye, isLegBye bool) error {
	bowlerRuns := 0
	if !isBye && !isLegBye {
		bowlerRuns = req.RunsScored + req.ExtrasRuns
		if isWide || isNoBall {
			bowlerRuns++ // penalty
		}
	}

	bowlerWickets := 0
	if req.IsWicket && req.DismissalType != nil && *req.DismissalType != "runout" && *req.DismissalType != "retired_hurt" {
		bowlerWickets = 1
	}

	//Fetch current overs count
	var currentOvers float64
	err := tx.Get(&currentOvers, `
		SELECT COALESCE((SELECT overs_bowled FROM player_match_stats WHERE match_id = $1 AND player_id = $2), 0)`,
		matchID, req.BowlerID)
	if err != nil {
		return err
	}

	//check for new overs
	var newOvers float64
	if isLegal {
		completedOvers := int(currentOvers)
		balls := int(math.Round((currentOvers - float64(completedOvers)) * 10))

		totalBalls := completedOvers*6 + balls + 1
		newOvers = float64(totalBalls/6) + float64(totalBalls%6)*0.1
	} else {
		newOvers = currentOvers
	}

	
	query := `
		INSERT INTO player_match_stats (match_id, player_id, runs_given, wickets_taken, maiden_overs, overs_bowled)
		VALUES ($1, $2, $3, $4, 0, $5)
		ON CONFLICT (match_id, player_id) DO UPDATE SET
			runs_given    = player_match_stats.runs_given    + EXCLUDED.runs_given,
			wickets_taken = player_match_stats.wickets_taken + EXCLUDED.wickets_taken,
			overs_bowled  = $5,
			updated_at    = NOW()`

	_, err = tx.Exec(query, matchID, req.BowlerID, bowlerRuns, bowlerWickets, newOvers)
	return err
}

// UpdateDismissedPlayerStats
func UpdateDismissedPlayerStatsTx(tx *sqlx.Tx, matchID string, req models.RecordBallRequest) error {
	if req.DismissedPlayerID == nil {
		return nil
	}

	var bowlerIDVal *string
	if req.DismissalType != nil && *req.DismissalType != "runout" && *req.DismissalType != "retired_hurt" {
		bowlerIDVal = &req.BowlerID
	}

	var fielderIDVal *string
	if req.FielderID != nil && *req.FielderID != "" {
		fielderIDVal = req.FielderID
	}

	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, player_id, is_not_out, dismissal_type, dismissed_by, fielder_id)
		VALUES ($1, $2, FALSE, $3::dismissal_type_enum, $4, $5)
		ON CONFLICT (match_id, player_id) DO UPDATE SET
			is_not_out     = FALSE,
			dismissal_type = EXCLUDED.dismissal_type,
			dismissed_by   = EXCLUDED.dismissed_by,
			fielder_id     = EXCLUDED.fielder_id,
			updated_at     = NOW()`,
		matchID, *req.DismissedPlayerID, req.DismissalType, bowlerIDVal, fielderIDVal,
	)
	return err
}

// FetchPlayerMatchStatsSummary
func FetchPlayerMatchStatsSummaryTx(tx *sqlx.Tx, matchID string, playerID string) (*models.PlayerStatsSummary, error) {
	var stats models.PlayerStatsSummary
	err := tx.Get(&stats, `
		SELECT player_id, runs_scored, balls_faced, fours, sixes, COALESCE(is_not_out, FALSE) as is_not_out, dismissal_type, dismissed_by, runs_given, wickets_taken, overs_bowled
		FROM player_match_stats
		WHERE match_id = $1 AND player_id = $2`,
		matchID, playerID,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

//splits them into batting and bowling teams inc common
func FetchInningsPlayers(inningsID string) (*models.InningsPlayersDetails, error) {
	var innings struct {
		MatchID            string  `db:"match_id"`
		InningsNumber      int     `db:"innings_number"`
		BattingTeamID      string  `db:"batting_team_id"`
		BowlingTeamID      string  `db:"bowling_team_id"`
		BattingTeamName    string  `db:"batting_team_name"`
		BowlingTeamName    string  `db:"bowling_team_name"`
		MatchStatus        string  `db:"match_status"`
		ActiveStrikerID    *string `db:"active_striker_id"`
		ActiveNonStrikerID *string `db:"active_non_striker_id"`
		ActiveBowlerID     *string `db:"active_bowler_id"`
		TotalRuns          int     `db:"total_runs"`
		TotalWickets       int     `db:"total_wickets"`
		TotalOvers         float64 `db:"total_overs"`
		TotalOversLimit    int     `db:"total_overs_limit"`
		TossWinnerTeamID   *string `db:"toss_winner_team_id"`
		TossDecision       *string `db:"toss_decision"`
		CommonPlayerID     *string `db:"common_player_id"`
	}
	err := db.KhiladiDb.Get(&innings, `
		SELECT 
			i.match_id, 
			i.innings_number,
			i.batting_team_id, 
			i.bowling_team_id,
			t1.team_name AS batting_team_name,
			t2.team_name AS bowling_team_name,
			m.status AS match_status,
			i.active_striker_id,
			i.active_non_striker_id,
			i.active_bowler_id,
			i.total_runs,
			i.total_wickets,
			i.total_overs,
			m.total_overs AS total_overs_limit,
			m.toss_winner_team_id,
			m.toss_decision,
			m.common_player_id
		FROM innings i
		JOIN teams t1 ON i.batting_team_id = t1.id
		JOIN teams t2 ON i.bowling_team_id = t2.id
		JOIN matches m ON i.match_id = m.id
		WHERE i.id = $1`, inningsID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch innings details: %w", err)
	}

	type MatchPlayerRow struct {
		models.MatchRosterPlayer
		TeamID string `db:"team_id"`
	}

	var rows []MatchPlayerRow
	query := `
		SELECT 
			p.id, 
			u.name AS player_name, 
			u.phone_number, 
			p.user_id, 
			p.role, 
			p.batting_style, 
			p.bowling_style,
			mp.team_id
		FROM match_players mp
		JOIN player_stats p ON mp.player_id = p.id
		JOIN users u ON p.user_id = u.id
		WHERE mp.match_id = $1
		ORDER BY u.name ASC
	`
	err = db.KhiladiDb.Select(&rows, query, innings.MatchID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch match players: %w", err)
	}

	battingPlayers := []models.MatchRosterPlayer{}
	bowlingPlayers := []models.MatchRosterPlayer{}

	seenBatting := make(map[string]bool)
	seenBowling := make(map[string]bool)

	for _, row := range rows {
		isCommon := innings.CommonPlayerID != nil && row.ID == *innings.CommonPlayerID
		if isCommon {
			if !seenBatting[row.ID] {
				battingPlayers = append(battingPlayers, row.MatchRosterPlayer)
				seenBatting[row.ID] = true
			}
			if !seenBowling[row.ID] {
				bowlingPlayers = append(bowlingPlayers, row.MatchRosterPlayer)
				seenBowling[row.ID] = true
			}
		} else if strings.EqualFold(row.TeamID, innings.BattingTeamID) {
			if !seenBatting[row.ID] {
				battingPlayers = append(battingPlayers, row.MatchRosterPlayer)
				seenBatting[row.ID] = true
			}
		} else if strings.EqualFold(row.TeamID, innings.BowlingTeamID) {
			if !seenBowling[row.ID] {
				bowlingPlayers = append(bowlingPlayers, row.MatchRosterPlayer)
				seenBowling[row.ID] = true
			}
		}
	}

	bowlStats, err := FetchBowlingStats(innings.MatchID, inningsID, innings.BowlingTeamID)
	if err != nil {
		bowlStats = []models.BowlStat{}
	}

	details := &models.InningsPlayersDetails{
		MatchID:            innings.MatchID,
		InningsNumber:      innings.InningsNumber,
		MatchStatus:        innings.MatchStatus,
		BattingTeamID:      innings.BattingTeamID,
		BowlingTeamID:      innings.BowlingTeamID,
		BattingTeamName:    innings.BattingTeamName,
		BowlingTeamName:    innings.BowlingTeamName,
		BattingPlayers:     battingPlayers,
		BowlingPlayers:     bowlingPlayers,
		ActiveStrikerID:    innings.ActiveStrikerID,
		ActiveNonStrikerID: innings.ActiveNonStrikerID,
		ActiveBowlerID:     innings.ActiveBowlerID,
		TotalRuns:          innings.TotalRuns,
		TotalWickets:       innings.TotalWickets,
		TotalOvers:         innings.TotalOvers,
		TotalOversLimit:    innings.TotalOversLimit,
		TossWinnerTeamID:   innings.TossWinnerTeamID,
		TossDecision:       innings.TossDecision,
		BowlerStats:        bowlStats,
	}

	return details, nil
}

const insertInningsQuery = `
	INSERT INTO innings (
		match_id,
		innings_number,
		batting_team_id,
		bowling_team_id,
		status,
		active_striker_id,
		active_non_striker_id,
		active_bowler_id
	)
	VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, NULLIF($7, '')::uuid, NULLIF($8, '')::uuid)
	RETURNING id
`

// CreateInningsTx (1st inning with  match creation)
func CreateInningsTx(tx *sqlx.Tx, matchID string, inningsNumber int, battingTeamID, bowlingTeamID string, status,strikerID,nonStrikerID,bowlerID string) (string, error) {
	var inningsID string
	err := tx.Get(&inningsID, `
		INSERT INTO innings (match_id, innings_number, batting_team_id, bowling_team_id, status,active_striker_id,active_non_striker_id,active_bowler_id)
		VALUES ($1, $2, $3, $4, $5,$6,$7,$8)
		RETURNING id`, matchID, inningsNumber, battingTeamID, bowlingTeamID, status,strikerID,nonStrikerID,bowlerID)
	return inningsID, err
}

// CreateInnings (2nd inning )
func CreateInnings(matchID string, inningsNumber int, battingTeamID, bowlingTeamID string, status string, strikerID, nonStrikerID, bowlerID string) (string, error) {
	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var inningsID string
	err = tx.Get(&inningsID, insertInningsQuery, matchID, inningsNumber, battingTeamID, bowlingTeamID, status, strikerID, nonStrikerID, bowlerID)
	if err != nil {
		return "", err
	}


	_, _ = tx.Exec(`
		UPDATE player_match_stats 
		SET is_not_out = TRUE 
		WHERE match_id = $1 
		AND dismissal_type IS NULL 
		AND player_id IN (
			SELECT player_id FROM match_players WHERE match_id = $1 AND team_id IN (
				SELECT batting_team_id FROM innings WHERE match_id = $1 AND innings_number < $2 AND status = 'live'
			)
		)`, matchID, inningsNumber)

	// Mark any previous innings of this match as completed and clear active players
	_, _ = tx.Exec(`
		UPDATE innings 
		SET status = 'completed', 
		    active_striker_id = NULL, 
		    active_non_striker_id = NULL, 
		    active_bowler_id = NULL, 
		    updated_at = NOW() 
		WHERE match_id = $1 AND innings_number < $2`, matchID, inningsNumber)

	// check for active striker non striker and bowler 
	if strikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id) 
			VALUES ($1, $2) 
			ON CONFLICT (match_id, player_id) DO NOTHING`, matchID, strikerID)
	}
	if nonStrikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id) 
			VALUES ($1, $2) 
			ON CONFLICT (match_id, player_id) DO NOTHING`, matchID, nonStrikerID)
	}

	if bowlerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id) 
			VALUES ($1, $2) 
			ON CONFLICT (match_id, player_id) DO NOTHING`, matchID, bowlerID)
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return inningsID, nil
}



// UpdateActivePlayers 
func UpdateActivePlayers(inningsID string, strikerID, nonStrikerID, bowlerID *string) error {
	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update active players in innings table
	_, err = tx.Exec(`
		UPDATE innings 
		SET active_striker_id = NULLIF($1, '')::uuid, 
		    active_non_striker_id = NULLIF($2, '')::uuid, 
		    active_bowler_id = NULLIF($3, '')::uuid 
		WHERE id = $4`,
		strikerID,
		nonStrikerID,
		bowlerID,
		inningsID,
	)

	if err != nil {
		return err
	}

	var matchID string
	err = tx.Get(&matchID, `SELECT match_id FROM innings WHERE id = $1`, inningsID)
	if err != nil {
		return err
	}

	if strikerID != nil && *strikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id)
			VALUES ($1, $2)
			ON CONFLICT (match_id, player_id) DO NOTHING`,
			matchID, *strikerID,
		)
	}

	if nonStrikerID != nil && *nonStrikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id)
			VALUES ($1, $2)
			ON CONFLICT (match_id, player_id) DO NOTHING`,
			matchID, *nonStrikerID,
		)
	}

	if bowlerID != nil && *bowlerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id)
			VALUES ($1, $2)
			ON CONFLICT (match_id, player_id) DO NOTHING`,
			matchID, *bowlerID,
		)
	}

	return tx.Commit()
}
