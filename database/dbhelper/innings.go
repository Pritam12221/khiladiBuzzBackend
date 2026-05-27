package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"

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

	// Insert the raw ball record
	if err = InsertBallTx(tx, inningsID, req); err != nil {
		return nil, fmt.Errorf("failed to insert ball: %w", err)
	}

	// Classify delivery
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

	// Update non-striker stats if active
	if req.NonStrikerID != nil && *req.NonStrikerID != "" {
		if err = UpdateNonStrikerStatsTx(tx, matchID, *req.NonStrikerID); err != nil {
			return nil, fmt.Errorf("failed to update non-striker stats: %w", err)
		}
	}

	//Update striker batting stats
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
	// 1. Fetch current overs count
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
		INSERT INTO player_match_stats (match_id, player_id, is_not_out)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (match_id, player_id) DO UPDATE SET
			is_not_out = COALESCE(player_match_stats.is_not_out, TRUE),
			updated_at = NOW()`,
		matchID, nonStrikerID,
	)
	return err
}

// UpdateStrikerStatsTx updates the active striker's batting statistics in the player_match_stats table.
func UpdateStrikerStatsTx(tx *sqlx.Tx, matchID string, strikerID string, runsScored int, isLegal, isBye, isLegBye bool) error {
	if isLegal {
		
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

		_, err := tx.Exec(`
			INSERT INTO player_match_stats (match_id, player_id, runs_scored, balls_faced, fours, sixes, is_not_out)
			VALUES ($1, $2, $3, 1, $4, $5, TRUE)
			ON CONFLICT (match_id, player_id) DO UPDATE SET
				runs_scored = player_match_stats.runs_scored + EXCLUDED.runs_scored,
				balls_faced = player_match_stats.balls_faced + 1,
				fours       = player_match_stats.fours       + EXCLUDED.fours,
				sixes       = player_match_stats.sixes       + EXCLUDED.sixes,
				is_not_out  = COALESCE(player_match_stats.is_not_out, TRUE),
				updated_at  = NOW()`,
			matchID, strikerID, runsForBatsman, foursInc, sixesInc,
		)
		return err
	}

	// Even on non-legal deliveries (e.g. wides), ensure striker is marked as not out
	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, player_id, is_not_out)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (match_id, player_id) DO UPDATE SET
			is_not_out = COALESCE(player_match_stats.is_not_out, TRUE),
			updated_at = NOW()`,
		matchID, strikerID,
	)
	return err
}

// UpdateBowlerStatsTx updates the bowler's bowling statistics in the player_match_stats table.
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

	// Compute new overs bowled using Go-level cricket math
	var newOvers float64
	if isLegal {
		completedOvers := int(currentOvers)
		balls := int(math.Round((currentOvers - float64(completedOvers)) * 10))

		totalBalls := completedOvers*6 + balls + 1
		newOvers = float64(totalBalls/6) + float64(totalBalls%6)*0.1
	} else {
		newOvers = currentOvers
	}

	// Perform upsert
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

// UpdateDismissedPlayerStatsTx updates the dismissed player's stats if a wicket fell.
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

// FetchPlayerMatchStatsSummaryTx fetches the match stats summary for a specific player.
func FetchPlayerMatchStatsSummaryTx(tx *sqlx.Tx, matchID string, playerID string) (*models.PlayerStatsSummary, error) {
	var stats models.PlayerStatsSummary
	err := tx.Get(&stats, `
		SELECT player_id, runs_scored, balls_faced, fours, sixes, is_not_out, dismissal_type, dismissed_by, runs_given, wickets_taken, overs_bowled
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
func FetchInningsPlayers(inningsID string) ([]models.Player, []models.Player, string, string, string, string, string, error) {
	var innings struct {
		MatchID         string `db:"match_id"`
		BattingTeamID    string `db:"batting_team_id"`
		BowlingTeamID    string `db:"bowling_team_id"`
		BattingTeamName  string `db:"batting_team_name"`
		BowlingTeamName  string `db:"bowling_team_name"`
	}
	err := db.KhiladiDb.Get(&innings, `
		SELECT 
			i.match_id, 
			i.batting_team_id, 
			i.bowling_team_id,
			t1.team_name AS batting_team_name,
			t2.team_name AS bowling_team_name
		FROM innings i
		JOIN teams t1 ON i.batting_team_id = t1.id
		JOIN teams t2 ON i.bowling_team_id = t2.id
		WHERE i.id = $1`, inningsID)
	if err != nil {
		return nil, nil, "", "", "", "", "", fmt.Errorf("failed to fetch innings details: %w", err)
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
		return nil, nil, "", "", "", "", "", fmt.Errorf("failed to fetch match players: %w", err)
	}

	battingPlayers := []models.Player{}
	bowlingPlayers := []models.Player{}

	for _, row := range rows {
		if row.TeamID == innings.BattingTeamID {
			battingPlayers = append(battingPlayers, row.Player)
		} else if row.TeamID == innings.BowlingTeamID {
			bowlingPlayers = append(bowlingPlayers, row.Player)
		}
	}

	return battingPlayers, bowlingPlayers, innings.BattingTeamName, innings.BowlingTeamName, innings.BattingTeamID, innings.BowlingTeamID, innings.MatchID, nil
}

const insertInningsQuery = `
	INSERT INTO innings (
		match_id,
		innings_number,
		batting_team_id,
		bowling_team_id,
		status
	)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
`

// CreateInningsTx creates an innings record using the provided transaction
func CreateInningsTx(tx *sqlx.Tx, matchID string, inningsNumber int, battingTeamID, bowlingTeamID string, status string) (string, error) {
	var inningsID string
	err := tx.Get(&inningsID, insertInningsQuery, matchID, inningsNumber, battingTeamID, bowlingTeamID, status)
	return inningsID, err
}

// CreateInnings creates an innings record in the database
func CreateInnings(matchID string, inningsNumber int, battingTeamID, bowlingTeamID string, status string) (string, error) {
	var inningsID string
	err := db.KhiladiDb.Get(&inningsID, insertInningsQuery, matchID, inningsNumber, battingTeamID, bowlingTeamID, status)
	return inningsID, err
}

