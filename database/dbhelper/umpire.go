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

//record ball by ball
func RecordBall(inningsID string, matchID string, req models.RecordBallRequest) (*models.RecordBallResponseDetails, error) {
	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var isCompleted bool
	err = tx.Get(&isCompleted, `SELECT EXISTS(SELECT 1 FROM matches WHERE id = $1 AND status = 'completed')`, matchID)
	if err == nil && isCompleted {
		return nil, fmt.Errorf("validation error: cannot record ball, match is already completed")
	}
	
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