package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"
	"strings"

	"github.com/jmoiron/sqlx"
)


// InsertBallTx inserts a new ball record in the database using the provided transaction
func InsertBallTx(tx *sqlx.Tx, inningsID string, req models.RecordBallRequest) error {
	var nonStrikerID *string
	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		nonStrikerID = req.NonStrikerID
	}

	_, err := tx.Exec(`
		INSERT INTO balls (
			innings_id, over_number, ball_number,
			striker_id, non_striker_id, bowler_id,
			runs_scored, extras_runs, extra_type,
			is_wicket, dismissal_type, dismissed_player_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::extra_type_enum, $10, $11::dismissal_type_enum, $12)`,
		inningsID,
		req.OverNumber, req.BallNumber,
		req.StrikerID, nonStrikerID, req.BowlerID,
		req.RunsScored, req.ExtrasRuns, req.ExtraType,
		req.IsWicket, req.DismissalType, req.DismissedPlayerID,
	)
	return err
}


func RecordBall(inningsID string, matchID string, req models.RecordBallRequest) (*models.RecordBallResponseDetails, error) {
	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Validate that the striker and non-striker if they are out or not
	var strikerDismissed bool
	err = tx.Get(&strikerDismissed, `
		SELECT EXISTS (
			SELECT 1 FROM balls 
			WHERE innings_id = $1 AND dismissed_player_id = $2 AND is_wicket = TRUE
		)
	`, inningsID, req.StrikerID)
	if err == nil && strikerDismissed {
		return nil, fmt.Errorf("validation error: striker %s has already been dismissed in this innings", req.StrikerID)
	}

	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		var nonStrikerDismissed bool
		err = tx.Get(&nonStrikerDismissed, `
			SELECT EXISTS (
				SELECT 1 FROM balls 
				WHERE innings_id = $1 AND dismissed_player_id = $2 AND is_wicket = TRUE
			)
		`, inningsID, *req.NonStrikerID)
		if err == nil && nonStrikerDismissed {
			return nil, fmt.Errorf("validation error: non-striker %s has already been dismissed in this innings", *req.NonStrikerID)
		}
	}

	// Check if the previous ball was a no-ball (Free Hit)
	var prevExtraType *string
	err = tx.Get(&prevExtraType, `
		SELECT extra_type FROM balls 
		WHERE innings_id = $1 AND (extra_type IS NULL OR extra_type != 'wide')
		ORDER BY over_number DESC, ball_number DESC, created_at DESC 
		LIMIT 1
	`, inningsID)
	
	isFreeHit := err == nil && prevExtraType != nil && *prevExtraType == "no_ball"
	isCurrentNoBall := req.ExtraType != nil && *req.ExtraType == "no_ball"

	if (isFreeHit || isCurrentNoBall) && req.IsWicket && req.DismissalType != nil {
		dtype := strings.ToLower(*req.DismissalType)
		if dtype != "runout" && dtype != "retired_hurt" {
			return nil, fmt.Errorf("validation error: on a no-ball or free hit, a batsman can only be out by run out or retired hurt")
		}
	}

	// Insert the raw ball record
	if err = InsertBallTx(tx, inningsID, req); err != nil {
		return nil, fmt.Errorf("failed to insert ball: %w", err)
	}

	// extra type
	isWide    := req.ExtraType != nil && *req.ExtraType == "wide"
	isNoBall  := req.ExtraType != nil && *req.ExtraType == "no_ball"
	isBye     := req.ExtraType != nil && *req.ExtraType == "bye"
	isLegBye  := req.ExtraType != nil && *req.ExtraType == "leg_bye"
	isLegal   := !isWide && !isNoBall 

	// Calculate innings run increment
	inningsRuns := req.RunsScored + req.ExtrasRuns
	if isWide || isNoBall {
		inningsRuns++ 
	}

	wicketDelta := 0
	if req.IsWicket {
		wicketDelta = 1
	}

	// Update innings total runs, wickets, and overs
	if err = UpdateInningsStatsTx(tx, inningsID, inningsRuns, wicketDelta, isLegal); err != nil {
		return nil, fmt.Errorf("failed to update innings: %w", err)
	}

	// Fetch current innings details (needed for batting team ID)
	var currentInnings struct {
		InningsNumber int     `db:"innings_number"`
		TotalRuns     int     `db:"total_runs"`
		TotalWickets  int     `db:"total_wickets"`
		TotalOvers    float64 `db:"total_overs"`
		BattingTeamID string  `db:"batting_team_id"`
		BowlingTeamID string  `db:"bowling_team_id"`
	}
	err = tx.Get(&currentInnings, `SELECT innings_number, total_runs, total_wickets, total_overs, batting_team_id, bowling_team_id FROM innings WHERE id = $1`, inningsID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current innings: %w", err)
	}

	// Update active players in the innings table in real-time
	var strikerVal *string = &req.StrikerID
	var nonStrikerVal *string
	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		nonStrikerVal = req.NonStrikerID
	}

	if req.IsWicket && req.DismissedPlayerID != nil {
		if *req.DismissedPlayerID == req.StrikerID {
			strikerVal = nil
		} else if nonStrikerVal != nil && *req.DismissedPlayerID == *nonStrikerVal {
			nonStrikerVal = nil
		}
	}

	_, _ = tx.Exec(`
		UPDATE innings 
		SET active_striker_id = $1, 
		    active_non_striker_id = $2, 
		    active_bowler_id = $3 
		WHERE id = $4`, strikerVal, nonStrikerVal, req.BowlerID, inningsID)


	// Update non-striker stats if active
	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		if err = UpdateNonStrikerStatsTx(tx, matchID, *req.NonStrikerID); err != nil {
			return nil, fmt.Errorf("failed to update non-striker stats: %w", err)
		}
	}

	// Update striker batting stats (passing isLegal)
	if err = UpdateStrikerStatsTx(tx, matchID, req.StrikerID, req.RunsScored, isLegal, isBye, isLegBye); err != nil {
		return nil, fmt.Errorf("failed to update striker stats: %w", err)
	}

	// Update bowler bowling stats
	if err = UpdateBowlerStatsTx(tx, matchID, req, isLegal, isWide, isNoBall, isBye, isLegBye); err != nil {
		return nil, fmt.Errorf("failed to update bowler stats: %w", err)
	}

	//  Update dismissed player stats if a wicket fell
	if req.IsWicket && req.DismissedPlayerID != nil {
		if err = UpdateDismissedPlayerStatsTx(tx, matchID, req); err != nil {
			return nil, fmt.Errorf("failed to update dismissed player stats: %w", err)
		}
	}

	// Check and update Innings/Match Completion and Winner
	var totalOversLimit int
	err = tx.Get(&totalOversLimit, `SELECT total_overs FROM matches WHERE id = $1`, matchID)
	if err == nil {
		completedOvers := int(currentInnings.TotalOvers)
		balls := int(math.Round((currentInnings.TotalOvers - float64(completedOvers)) * 10))
		totalBalls := completedOvers*6 + balls

		var teamSize int
		err = tx.Get(&teamSize, `SELECT COUNT(*) FROM match_players WHERE match_id = $1 AND team_id = $2`, matchID, currentInnings.BattingTeamID)
		if err != nil {
			teamSize = 11 // fallback size
		}

		isOversFinished := totalBalls >= totalOversLimit*6
		isAllOut := currentInnings.TotalWickets == 11 || currentInnings.TotalWickets == teamSize

		// Innings 1 Completion Check
		if currentInnings.InningsNumber == 1 {
			if isOversFinished || isAllOut {
				// 1. Finalize surviving batsmen for Innings 1 (while active_striker_id/active_non_striker_id are still set)
				_, _ = tx.Exec(`
					UPDATE player_match_stats 
					SET is_not_out = TRUE 
					WHERE match_id = $1 
					AND dismissal_type IS NULL 
					AND (
						runs_scored > 0 
						OR balls_faced > 0 
						OR player_id = (SELECT active_striker_id FROM innings WHERE id = $3)
						OR player_id = (SELECT active_non_striker_id FROM innings WHERE id = $3)
					)
					AND player_id IN (
						SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2
					)`, matchID, currentInnings.BattingTeamID, inningsID)

				// 2. Clear active players in innings table and mark as completed
				_, _ = tx.Exec(`
					UPDATE innings 
					SET status = 'completed', 
					    active_striker_id = NULL, 
					    active_non_striker_id = NULL, 
					    active_bowler_id = NULL, 
					    updated_at = NOW() 
					WHERE id = $1`, inningsID)
			}
		}

		// Innings 2 Completion Check
		if currentInnings.InningsNumber == 2 {
			var innings1Runs int
			err = tx.Get(&innings1Runs, `SELECT total_runs FROM innings WHERE match_id = $1 AND innings_number = 1`, matchID)
			if err == nil {
				chasedTarget := currentInnings.TotalRuns >= innings1Runs+1

				if chasedTarget {
					// Finalize surviving batsmen for Innings 2
					_, _ = tx.Exec(`
						UPDATE player_match_stats 
						SET is_not_out = TRUE 
						WHERE match_id = $1 
						AND dismissal_type IS NULL 
						AND (
							runs_scored > 0 
							OR balls_faced > 0 
							OR player_id = (SELECT active_striker_id FROM innings WHERE id = $3)
							OR player_id = (SELECT active_non_striker_id FROM innings WHERE id = $3)
						)
						AND player_id IN (
							SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2
						)`, matchID, currentInnings.BattingTeamID, inningsID)

					
					_, _ = tx.Exec(`
						UPDATE innings 
						SET status = 'completed', 
						    active_striker_id = NULL, 
						    active_non_striker_id = NULL, 
						    active_bowler_id = NULL, 
						    updated_at = NOW() 
						WHERE id = $1`, inningsID)
					_, err = tx.Exec(
						`UPDATE matches SET status = 'completed', winner_team_id = $1, updated_at = NOW() WHERE id = $2`, currentInnings.BattingTeamID, matchID)
				} else if isOversFinished || isAllOut {
					_, _ = tx.Exec(`
						UPDATE player_match_stats 
						SET is_not_out = TRUE 
						WHERE match_id = $1 
						AND dismissal_type IS NULL 
						AND (
							runs_scored > 0 
							OR balls_faced > 0 
							OR player_id = (SELECT active_striker_id FROM innings WHERE id = $3)
							OR player_id = (SELECT active_non_striker_id FROM innings WHERE id = $3)
						)
						AND player_id IN (
							SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2
						)`, matchID, currentInnings.BattingTeamID, inningsID)

					
					_, _ = tx.Exec(`
						UPDATE innings 
						SET status = 'completed', 
						    active_striker_id = NULL, 
						    active_non_striker_id = NULL, 
						    active_bowler_id = NULL, 
						    updated_at = NOW() 
						WHERE id = $1`, inningsID)
					var winnerTeamID *string
					if currentInnings.TotalRuns < innings1Runs {
						winnerTeamID = &currentInnings.BowlingTeamID
					}
					_, err = tx.Exec(`UPDATE matches SET status = 'completed', winner_team_id = $1, updated_at = NOW() WHERE id = $2`, winnerTeamID, matchID)
				}
			}
		}
	}

	// Fetch latest statistics for the response payload
	var result models.RecordBallResponseDetails

	if strikerStats, err := FetchPlayerMatchStatsSummaryTx(tx, matchID, req.StrikerID); err == nil {
		result.Striker = strikerStats
	}
	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		if nonStrikerStats, err := FetchPlayerMatchStatsSummaryTx(tx, matchID, *req.NonStrikerID); err == nil {
			result.NonStriker = nonStrikerStats
		}
	}
	if bowlerStats, err := FetchPlayerMatchStatsSummaryTx(tx, matchID, req.BowlerID); err == nil {
		result.Bowler = bowlerStats
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &result, nil
}


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

// FetchInningsPlayers fetches all match players for a given innings and splits them into batting and bowling rosters.
func FetchInningsPlayers(inningsID string) ([]models.Player, []models.Player, string, string, string, string, string, int, string, error) {
	var innings struct {
		MatchID          string `db:"match_id"`
		InningsNumber    int    `db:"innings_number"`
		BattingTeamID    string `db:"batting_team_id"`
		BowlingTeamID    string `db:"bowling_team_id"`
		BattingTeamName  string `db:"batting_team_name"`
		BowlingTeamName  string `db:"bowling_team_name"`
		MatchStatus      string `db:"match_status"`
	}
	err := db.KhiladiDb.Get(&innings, `
		SELECT 
			i.match_id, 
			i.innings_number,
			i.batting_team_id, 
			i.bowling_team_id,
			t1.team_name AS batting_team_name,
			t2.team_name AS bowling_team_name,
			m.status AS match_status
		FROM innings i
		JOIN teams t1 ON i.batting_team_id = t1.id
		JOIN teams t2 ON i.bowling_team_id = t2.id
		JOIN matches m ON i.match_id = m.id
		WHERE i.id = $1`, inningsID)
	if err != nil {
		return nil, nil, "", "", "", "", "", 0, "", fmt.Errorf("failed to fetch innings details: %w", err)
	}

	type MatchPlayerRow struct {
		models.Player
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
		return nil, nil, "", "", "", "", "", 0, "", fmt.Errorf("failed to fetch match players: %w", err)
	}

	battingPlayers := []models.Player{}
	bowlingPlayers := []models.Player{}

	for _, row := range rows {
		if strings.EqualFold(row.TeamID, innings.BattingTeamID) {
			battingPlayers = append(battingPlayers, row.Player)
		} else if strings.EqualFold(row.TeamID, innings.BowlingTeamID) {
			bowlingPlayers = append(bowlingPlayers, row.Player)
		}
	}

	return battingPlayers, bowlingPlayers, innings.BattingTeamName, innings.BowlingTeamName, innings.BattingTeamID, innings.BowlingTeamID, innings.MatchID, innings.InningsNumber, innings.MatchStatus, nil
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

// CreateInningsTx creates an innings record using the provided transaction
func CreateInningsTx(tx *sqlx.Tx, matchID string, inningsNumber int, battingTeamID, bowlingTeamID string, status string) (string, error) {
	var inningsID string
	// Backward compatible CreateInningsTx (does not set active players, but we can set them via RecordBall or separate update)
	err := tx.Get(&inningsID, `
		INSERT INTO innings (match_id, innings_number, batting_team_id, bowling_team_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, matchID, inningsNumber, battingTeamID, bowlingTeamID, status)
	return inningsID, err
}

// CreateInnings creates an innings record in the database and initializes opening batsmen and bowler
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

	// check for active striker non striker
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


	return tx.Commit()
}
