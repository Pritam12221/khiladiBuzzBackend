package dbhelper

import (
	"database/sql"
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"

)

func UndoLastBall(inningsID string) (*models.InningsPlayersDetails, error) {
	tx, err := db.KhiladiDb.Beginx()


	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch the last ball recorded in the innings
	var lastBall struct {
		ID                string  `db:"id"`
		StrikerID         string  `db:"striker_id"`
		NonStrikerID      *string `db:"non_striker_id"`
		BowlerID          string  `db:"bowler_id"`
		RunsScored        int     `db:"runs_scored"`
		ExtrasRuns        int     `db:"extras_runs"`
		ExtraType         *string `db:"extra_type"`
		IsWicket          bool    `db:"is_wicket"`
		DismissalType     *string `db:"dismissal_type"`
		DismissedPlayerID *string `db:"dismissed_player_id"`
	}

	err = tx.Get(&lastBall, `
		SELECT id, striker_id, non_striker_id, bowler_id, runs_scored, extras_runs, extra_type, is_wicket, dismissal_type, dismissed_player_id
		FROM balls
		WHERE innings_id = $1
		ORDER BY over_number DESC, ball_number DESC, created_at DESC
		LIMIT 1`, inningsID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no balls found in this innings to undo")
		}
		return nil, fmt.Errorf("failed to fetch last ball: %w", err)
	}

	// Fetch current innings details to know the lst ball in an inning
	var innings struct {
		MatchID       string  `db:"match_id"`
		InningsNumber int     `db:"innings_number"`
		TotalRuns     int     `db:"total_runs"`
		TotalWickets  int     `db:"total_wickets"`
		TotalOvers    float64 `db:"total_overs"`
	}
	err = tx.Get(&innings, `SELECT match_id, innings_number, total_runs, total_wickets, total_overs FROM innings WHERE id = $1`, inningsID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch innings: %w", err)
	}

	//  Delete the last ball
	_, err = tx.Exec(`DELETE FROM balls WHERE id = $1`, lastBall.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete ball: %w", err)
	}

	// Calculate stats decrement
	isWide := lastBall.ExtraType != nil && *lastBall.ExtraType == "wide"
	isNoBall := lastBall.ExtraType != nil && *lastBall.ExtraType == "no_ball"
	isBye := lastBall.ExtraType != nil && *lastBall.ExtraType == "bye"
	isLegBye := lastBall.ExtraType != nil && *lastBall.ExtraType == "leg_bye"
	isLegal := !isWide && !isNoBall

	ballRuns := lastBall.RunsScored + lastBall.ExtrasRuns
	if isWide || isNoBall {
		ballRuns++ // penalty run
	}

	newRuns := innings.TotalRuns - ballRuns
	if newRuns < 0 {
		newRuns = 0
	}
	newWickets := innings.TotalWickets
	if lastBall.IsWicket {
		newWickets--
	}
	if newWickets < 0 {
		newWickets = 0
	}

	// Revert innings overs count
	var newOvers float64
	var totalLegalBalls int
	if isLegal {
		currentOvers := innings.TotalOvers
		completedOvers := int(currentOvers)
		balls := int(math.Round((currentOvers - float64(completedOvers)) * 10))

		totalLegalBalls = completedOvers*6 + balls - 1
		if totalLegalBalls < 0 {
			totalLegalBalls = 0
		}
		newOvers = float64(totalLegalBalls/6) + float64(totalLegalBalls%6)*0.1
	} else {
		newOvers = innings.TotalOvers
		completedOvers := int(newOvers)
		balls := int(math.Round((newOvers - float64(completedOvers)) * 10))
		totalLegalBalls = completedOvers*6 + balls
	}

	// Revert innings stats on last ball
	_, err = tx.Exec(`
		UPDATE innings SET
			total_runs    = $1,
			total_wickets = $2,
			total_overs   = $3,
			status        = 'live',
			updated_at    = NOW()
		WHERE id = $4`,
		newRuns, newWickets, newOvers, inningsID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update innings stats: %w", err)
	}

	// if match complete on last bll revert back the changes
	var match struct {
		Status       string  `db:"status"`
		WinnerTeamID *string `db:"winner_team_id"`
	}
	err = tx.Get(&match, `SELECT status, winner_team_id FROM matches WHERE id = $1`, innings.MatchID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch match: %w", err)
	}

	//if macth completed in last we also need to subtract it from overall player stats
	if match.Status == "completed" {

		_, err = tx.Exec(`
			WITH match_batting AS (
				SELECT 
					player_id,
					SUM(runs_scored) as total_runs,
					SUM(balls_faced) as total_balls_faced,
					SUM(fours) as total_fours,
					SUM(sixes) as total_sixes,
					COALESCE(MAX(CASE WHEN is_not_out THEN 1 ELSE 0 END), 0) = 1 as is_not_out
				FROM player_match_stats
				WHERE match_id = $1
				GROUP BY player_id
			),
			match_bowling AS (
				SELECT 
					b.bowler_id as player_id,
					COUNT(*) as total_balls_bowled
				FROM balls b
				JOIN innings i ON b.innings_id = i.id
				WHERE i.match_id = $1
				  AND (b.extra_type IS NULL OR b.extra_type::text NOT IN ('wide', 'no_ball'))
				GROUP BY b.bowler_id
			),
			match_bowling_stats AS (
				SELECT 
					player_id,
					SUM(runs_given) as total_runs_given,
					SUM(wickets_taken) as total_wickets,
					SUM(maiden_overs) as total_maidens
				FROM player_match_stats
				WHERE match_id = $1
				GROUP BY player_id
			)
			UPDATE player_stats ps
			SET
				career_matches = ps.career_matches - 1,
				career_wins = ps.career_wins - CASE WHEN $2::uuid IS NOT NULL AND ps.id IN (
					SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2::uuid
				) THEN 1 ELSE 0 END,
				career_runs = ps.career_runs - COALESCE(mb.total_runs, 0),
				career_balls_faced = ps.career_balls_faced - COALESCE(mb.total_balls_faced, 0),
				career_fours = ps.career_fours - COALESCE(mb.total_fours, 0),
				career_sixes = ps.career_sixes - COALESCE(mb.total_sixes, 0),
				career_runs_given = ps.career_runs_given - COALESCE(mbs.total_runs_given, 0),
				career_wickets = ps.career_wickets - COALESCE(mbs.total_wickets, 0),
				career_balls_bowled = ps.career_balls_bowled - COALESCE(mbol.total_balls_bowled, 0),
				career_ducks = ps.career_ducks - CASE WHEN mb.total_runs = 0 AND NOT mb.is_not_out AND mb.total_balls_faced > 0 THEN 1 ELSE 0 END,
				career_fifties = ps.career_fifties - CASE WHEN mb.total_runs >= 50 AND mb.total_runs < 100 THEN 1 ELSE 0 END,
				career_hundreds = ps.career_hundreds - CASE WHEN mb.total_runs >= 100 THEN 1 ELSE 0 END,
				career_highest_score = COALESCE((
					SELECT MAX(runs_scored) 
					FROM player_match_stats 
					WHERE player_id = ps.id AND match_id != $1
				), 0),
				career_maidens = ps.career_maidens - COALESCE(mbs.total_maidens, 0),
				career_highest_wickets = COALESCE((
					SELECT MAX(wickets_taken) 
					FROM player_match_stats 
					WHERE player_id = ps.id AND match_id != $1
				), 0),
				updated_at = NOW()
			FROM (
				SELECT DISTINCT player_id FROM match_players WHERE match_id = $1
			) mp
			LEFT JOIN match_batting mb ON mp.player_id = mb.player_id
			LEFT JOIN match_bowling mbol ON mp.player_id = mbol.player_id
			LEFT JOIN match_bowling_stats mbs ON mp.player_id = mbs.player_id
			WHERE ps.id = mp.player_id;`, innings.MatchID, match.WinnerTeamID)
		if err != nil {
			return nil, fmt.Errorf("failed to rollback career stats: %w", err)
		}

		// Set match status back to live
		_, err = tx.Exec(`UPDATE matches SET status = 'live', winner_team_id = NULL, updated_at = NOW() WHERE id = $1`, innings.MatchID)
		if err != nil {
			return nil, fmt.Errorf("failed to revert match status: %w", err)
		}
	}

	// Revert player match stats for player
	runsForBatsman := lastBall.RunsScored
	if isBye || isLegBye {
		runsForBatsman = 0
	}
	foursDec, sixesDec := 0, 0
	switch runsForBatsman {
		case 4:
		foursDec = 1
		case 6:
		sixesDec = 1
	}
	ballsFacedDec := 0
	if isLegal {
		ballsFacedDec = 1
	}

	_, err = tx.Exec(`
		UPDATE player_match_stats SET
			runs_scored = COALESCE(runs_scored, 0) - $1,
			balls_faced = COALESCE(balls_faced, 0) - $2,
			fours       = COALESCE(fours, 0) - $3,
			sixes       = COALESCE(sixes, 0) - $4,
			updated_at  = NOW()
		WHERE innings_id = $5 AND player_id = $6`,
		runsForBatsman, ballsFacedDec, foursDec, sixesDec, inningsID, lastBall.StrikerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to revert striker match stats: %w", err)
	}

	// Revert bowler match stats
	bowlerRuns := 0
	if !isBye && !isLegBye {
		bowlerRuns = lastBall.RunsScored + lastBall.ExtrasRuns
		if isWide || isNoBall {
			bowlerRuns++
		}
	}
	bowlerWicketDec := 0
	if lastBall.IsWicket && lastBall.DismissalType != nil && *lastBall.DismissalType != "runout" && *lastBall.DismissalType != "retired_out" {
		bowlerWicketDec = 1
	}

	var currentBowlerOvers float64
	err = tx.Get(&currentBowlerOvers, `
		SELECT COALESCE(overs_bowled, 0) FROM player_match_stats WHERE innings_id = $1 AND player_id = $2`,
		inningsID, lastBall.BowlerID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to fetch bowler match overs: %w", err)
	}

	var newBowlerOvers float64
	if isLegal {
		completedOvers := int(currentBowlerOvers)
		balls := int(math.Round((currentBowlerOvers - float64(completedOvers)) * 10))

		totalBowlerBalls := completedOvers*6 + balls - 1
		if totalBowlerBalls < 0 {
			totalBowlerBalls = 0
		}
		newBowlerOvers = float64(totalBowlerBalls/6) + float64(totalBowlerBalls%6)*0.1
	} else {
		newBowlerOvers = currentBowlerOvers
	}

	_, err = tx.Exec(`
		UPDATE player_match_stats SET
			runs_given    = COALESCE(runs_given, 0) - $1,
			wickets_taken = COALESCE(wickets_taken, 0) - $2,
			overs_bowled  = $3,
			updated_at    = NOW()
		WHERE innings_id = $4 AND player_id = $5`,
		bowlerRuns, bowlerWicketDec, newBowlerOvers, inningsID, lastBall.BowlerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to revert bowler match stats: %w", err)
	}

	// Revert dismissed player to not out
	if lastBall.IsWicket && lastBall.DismissedPlayerID != nil {
		_, err = tx.Exec(`
			UPDATE player_match_stats SET
				is_not_out     = TRUE,
				dismissal_type = NULL,
				dismissed_by   = NULL,
				fielder_id     = NULL,
				updated_at     = NOW()
			WHERE innings_id = $1 AND player_id = $2`,
			inningsID, *lastBall.DismissedPlayerID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to revert dismissed player match stats: %w", err)
		}
	}

	// Restore active player strike rotation & handle bowler selection at over boundaries
	strikerArg := lastBall.StrikerID
	var nonStrikerArg *string = lastBall.NonStrikerID

	var bowlerArg *string
	// Set bowler to null to force bowler selection
	if totalLegalBalls%6 == 0 {
		bowlerArg = nil 
	} else {
		bowlerArg = &lastBall.BowlerID
	}

	_, err = tx.Exec(`
		UPDATE innings SET
			active_striker_id     = $1::uuid,
			active_non_striker_id = $2::uuid,
			active_bowler_id      = $3::uuid
		WHERE id = $4`,
		strikerArg, nonStrikerArg, bowlerArg, inningsID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to restore active players: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch and return fresh innings player details to frontend
	return FetchInningsPlayers(inningsID)
}
