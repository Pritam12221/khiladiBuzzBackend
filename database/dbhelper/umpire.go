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


func resolveNextActivePlayers(
	tx *sqlx.Tx,
	inningsID string,
	req models.RecordBallRequest,
) (strikerArg, nonStrikerArg, bowlerArg *string) {

	//current active plyers (for every bowl)
	var nextStrikerID *string = &req.StrikerID
	var nextNonStrikerID *string = req.NonStrikerID
	var nextBowlerID *string = &req.BowlerID

	// check if player is out or not
	if req.IsWicket && req.DismissedPlayerID != nil {
		if *req.DismissedPlayerID == req.StrikerID {
			nextStrikerID = nil
		} else if req.NonStrikerID != nil && *req.DismissedPlayerID == *req.NonStrikerID {
			nextNonStrikerID = nil
		}
	}

	// Strike rotation on odd runs
	runsRun := req.RunsScored
	if req.ExtraType != nil && (*req.ExtraType == "bye" || *req.ExtraType == "leg_bye" || *req.ExtraType == "wide") {
		runsRun = req.ExtrasRuns
	}
	if runsRun%2 != 0 && nextStrikerID != nil && nextNonStrikerID != nil {
		nextStrikerID, nextNonStrikerID = nextNonStrikerID, nextStrikerID
	}

	// check over based on legl bowls  (6)
	var validBalls int
	if err := tx.Get(&validBalls, `
		SELECT COUNT(*) FROM balls
		WHERE innings_id = $1
		  AND over_number = $2
		  AND (extra_type IS NULL OR (extra_type != 'wide' AND extra_type != 'no_ball'))
	`, inningsID, req.OverNumber); err == nil && validBalls >= 6 {
		// Batsmen change ends at the start of the next over
		if nextStrikerID != nil && nextNonStrikerID != nil {
			nextStrikerID, nextNonStrikerID = nextNonStrikerID, nextStrikerID
		}
		nextBowlerID = nil 
	}

	if nextStrikerID != nil && *nextStrikerID != "" {
		strikerArg = nextStrikerID
	}
	if nextNonStrikerID != nil && *nextNonStrikerID != "" {
		nonStrikerArg = nextNonStrikerID
	}
	if nextBowlerID != nil && *nextBowlerID != "" {
		bowlerArg = nextBowlerID
	}
	return
}


func fetchNextPlayerStats(tx *sqlx.Tx, matchID string, playerArg *string) *models.PlayerStatsSummary {
	if playerArg == nil || *playerArg == "" {
		return nil
	}
	stats, err := FetchPlayerMatchStatsSummaryTx(tx, matchID, *playerArg)
	if err != nil {
		return nil
	}
	return stats
}

// validateRecordBall checks if a ball can be legally recorded based on match status
func validateRecordBall(tx *sqlx.Tx, inningsID string, matchStatus string, req models.RecordBallRequest) error {
	if matchStatus == "completed" {
		return fmt.Errorf("validation error: cannot record ball, match is already completed")
	}
	
	// Validate that the striker and non-striker if they are out or not
	var strikerDismissed bool
	err := tx.Get(&strikerDismissed, `
		SELECT EXISTS (
			SELECT 1 FROM balls 
			WHERE innings_id = $1 AND dismissed_player_id = $2 AND is_wicket = TRUE
		)
	`, inningsID, req.StrikerID)
	if err == nil && strikerDismissed {
		return fmt.Errorf("validation error: striker %s has already been dismissed in this innings", req.StrikerID)
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
			return fmt.Errorf("validation error: non-striker %s has already been dismissed in this innings", *req.NonStrikerID)
		}
	}

	// Check if the previous ball was a no-ball 
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
			return fmt.Errorf("validation error: on a no-ball or free hit, a batsman can only be out by run out or retired hurt")
		}
	}
	return nil
}

// inning refinement after inning completion and checks
func checkAndUpdateInningsMatchCompletion(
	tx *sqlx.Tx,
	matchID string,
	inningsID string,
	inningsNumber int,
	totalRuns int,
	totalWickets int,
	totalOvers float64,
	battingTeamID string,
	bowlingTeamID string,
	totalOversLimit int,
) error {
	completedOvers := int(totalOvers)
	balls := int(math.Round((totalOvers - float64(completedOvers)) * 10))
	totalBalls := completedOvers*6 + balls

	var teamSize int
	err := tx.Get(&teamSize, `
		SELECT COUNT(DISTINCT player_id) 
		FROM (
			SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2
			UNION
			SELECT common_player_id AS player_id FROM matches WHERE id = $1 AND common_player_id IS NOT NULL
		) all_players`, matchID, battingTeamID)
	if err != nil {
		teamSize = 11 // fallback size
	}

	isOversFinished := totalBalls >= totalOversLimit*6
	isAllOut := totalWickets == 11 || totalWickets >= teamSize

	
	isInningsComplete := false
	isMatchComplete := false
	var winnerTeamID *string

	if inningsNumber == 1 {
		if isOversFinished || isAllOut {
			isInningsComplete = true
		}
	} else if inningsNumber == 2 {
		var innings1Runs int
		err = tx.Get(&innings1Runs, `SELECT total_runs FROM innings WHERE match_id = $1 AND innings_number = 1`, matchID)
		if err == nil {
			chasedTarget := totalRuns >= innings1Runs+1

			if chasedTarget {
				isInningsComplete = true
				isMatchComplete = true
				winnerTeamID = &battingTeamID
			} else if isOversFinished || isAllOut {
				isInningsComplete = true
				isMatchComplete = true
				if totalRuns < innings1Runs {
					winnerTeamID = &bowlingTeamID
				}
			}
		}
	}

	if isInningsComplete {
		// Finalize surviving batsmen
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
			)`, matchID, battingTeamID, inningsID)

		// Close innings
		_, _ = tx.Exec(`
			UPDATE innings 
			SET status = 'completed', 
			    active_striker_id = NULL, 
			    active_non_striker_id = NULL, 
			    active_bowler_id = NULL, 
			    updated_at = NOW() 
			WHERE id = $1`, inningsID)
	}

	if isMatchComplete {
		// Close match & record winner
		_, err = tx.Exec(`UPDATE matches SET status = 'completed', winner_team_id = $1, updated_at = NOW() WHERE id = $2`, winnerTeamID, matchID)
		if err != nil {
			return fmt.Errorf("failed to close match: %w", err)
		}

		// Bulk aggregate all match statistics and update career player_stats at once
		_, err = tx.Exec(`
			WITH match_batting AS (
				SELECT 
					player_id,
					SUM(runs_scored) as total_runs,
					SUM(balls_faced) as total_balls_faced,
					SUM(fours) as total_fours,
					SUM(sixes) as total_sixes
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
					SUM(wickets_taken) as total_wickets
				FROM player_match_stats
				WHERE match_id = $1
				GROUP BY player_id
			)
			UPDATE player_stats ps
			SET
				career_matches = ps.career_matches + 1,
				career_wins = ps.career_wins + CASE WHEN $2::uuid IS NOT NULL AND ps.id IN (
					SELECT player_id FROM match_players WHERE match_id = $1 AND team_id = $2::uuid
				) THEN 1 ELSE 0 END,
				career_runs = ps.career_runs + COALESCE(mb.total_runs, 0),
				career_balls_faced = ps.career_balls_faced + COALESCE(mb.total_balls_faced, 0),
				career_fours = ps.career_fours + COALESCE(mb.total_fours, 0),
				career_sixes = ps.career_sixes + COALESCE(mb.total_sixes, 0),
				career_runs_given = ps.career_runs_given + COALESCE(mbs.total_runs_given, 0),
				career_wickets = ps.career_wickets + COALESCE(mbs.total_wickets, 0),
				career_balls_bowled = ps.career_balls_bowled + COALESCE(mbol.total_balls_bowled, 0),
				updated_at = NOW()
			FROM (
				SELECT DISTINCT player_id FROM match_players WHERE match_id = $1
			) mp
			LEFT JOIN match_batting mb ON mp.player_id = mb.player_id
			LEFT JOIN match_bowling mbol ON mp.player_id = mbol.player_id
			LEFT JOIN match_bowling_stats mbs ON mp.player_id = mbs.player_id
			WHERE ps.id = mp.player_id;`, matchID, winnerTeamID)
		if err != nil {
			return fmt.Errorf("failed to update career stats: %w", err)
		}
	}

	return nil
}

//record ball by ball
func RecordBall(inningsID string, matchID string, req models.RecordBallRequest) (*models.RecordBallResponseDetails, error) {
	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var matchInfo struct {
		Status     string `db:"status"`
		TotalOvers int    `db:"total_overs"`
	}
	err = tx.Get(&matchInfo, `SELECT status, total_overs FROM matches WHERE id = $1`, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch match details: %w", err)
	}

	if err = validateRecordBall(tx, inningsID, matchInfo.Status, req); err != nil {
		return nil, err
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
	if err = checkAndUpdateInningsMatchCompletion(
		tx, matchID, inningsID,
		currentInnings.InningsNumber,
		currentInnings.TotalRuns,
		currentInnings.TotalWickets,
		currentInnings.TotalOvers,
		currentInnings.BattingTeamID,
		currentInnings.BowlingTeamID,
		matchInfo.TotalOvers,
	); err != nil {
		return nil, err
	}

	// Determine next active players and persist them (only if the innings is not completed yet)
	var strikerArg, nonStrikerArg, bowlerArg *string

	var isCompletedNow bool
	err = tx.Get(&isCompletedNow, `SELECT EXISTS(SELECT 1 FROM innings WHERE id = $1 AND status = 'completed')`, inningsID)
	if err == nil && !isCompletedNow {
		strikerArg, nonStrikerArg, bowlerArg = resolveNextActivePlayers(tx, inningsID, req)
	}

	_, err = tx.Exec(`
		UPDATE innings
		SET active_striker_id    = $1::uuid,
		    active_non_striker_id = $2::uuid,
		    active_bowler_id      = $3::uuid
		WHERE id = $4`,
		strikerArg, nonStrikerArg, bowlerArg, inningsID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update next active players: %w", err)
	}

	result := models.RecordBallResponseDetails{
		Striker:          fetchNextPlayerStats(tx, matchID, strikerArg),
		NonStriker:       fetchNextPlayerStats(tx, matchID, nonStrikerArg),
		Bowler:           fetchNextPlayerStats(tx, matchID, bowlerArg),
		NextStrikerID:    strikerArg,
		NextNonStrikerID: nonStrikerArg,
		NextBowlerID:     bowlerArg,
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &result, nil
}