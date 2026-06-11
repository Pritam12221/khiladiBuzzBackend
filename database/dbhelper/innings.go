package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"
	"strings"
	"khiladiBuzz/utils"

	"github.com/jmoiron/sqlx"
)


func UpdateInningsStatsTx(tx *sqlx.Tx, inningsID string, runsInc, wicketsInc int, isLegal bool) error {
	// Fetch current overs count
	var currentOvers float64
	err := tx.Get(&currentOvers, `SELECT total_overs FROM innings WHERE id = $1`, inningsID)
	if err != nil {
		return err
	}


		newOvers := utils.CalculateNewOvers(currentOvers, isLegal, 1)

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

// UpdateNonStrikerStatsTx ensures that it makes row for the non-striker player when he comes to bat
func UpdateNonStrikerStatsTx(tx *sqlx.Tx, inningsID string, nonStrikerID string) error {
	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, innings_id, player_id)
		VALUES ((SELECT match_id FROM innings WHERE id = $1), $1, $2)
		ON CONFLICT (innings_id, player_id) DO NOTHING`,
		inningsID, nonStrikerID,
	)
	return err
}

// UpdateStrikerStatsTx updates the active striker (strike rotations)
func UpdateStrikerStatsTx(tx *sqlx.Tx, inningsID string, strikerID string, runsScored int, isBallFaced, isBye, isLegBye bool) error {
	runsForBatsman := runsScored
	if isBye || isLegBye {
		runsForBatsman = 0
	}

	foursInc, sixesInc := 0, 0
	switch runsForBatsman {
	case 4:
		foursInc = 1
	case 6:
		sixesInc = 1
	}

	ballsFacedInc := 0
	if isBallFaced {
		ballsFacedInc = 1
	}

	_, err := tx.Exec(`
		INSERT INTO player_match_stats (match_id, innings_id, player_id, runs_scored, balls_faced, fours, sixes)
		VALUES ((SELECT match_id FROM innings WHERE id = $1), $1, $2, $3, $4, $5, $6)
		ON CONFLICT (innings_id, player_id) DO UPDATE SET
			runs_scored = player_match_stats.runs_scored + EXCLUDED.runs_scored,
			balls_faced = player_match_stats.balls_faced + EXCLUDED.balls_faced,
			fours       = player_match_stats.fours       + EXCLUDED.fours,
			sixes       = player_match_stats.sixes       + EXCLUDED.sixes,
			updated_at  = NOW()`,
		inningsID, strikerID, runsForBatsman, ballsFacedInc, foursInc, sixesInc,
	)
	return err
}

// UpdateBowlerStatsTx updates the bowler's bowling statistics in the player_match_stats table.
func UpdateBowlerStatsTx(tx *sqlx.Tx, inningsID string, req models.RecordBallRequest, isLegal, isWide, isNoBall, isBye, isLegBye bool) error {
	bowlerRuns := 0
	if !isBye && !isLegBye {
		bowlerRuns = req.RunsScored + req.ExtrasRuns
		if isWide || isNoBall {
			bowlerRuns++ // penalty
		}
	}

	bowlerWickets := 0
	if req.IsWicket && req.DismissalType != nil && *req.DismissalType != "runout" && *req.DismissalType != "retired_out" {
		bowlerWickets = 1
	}

	//Fetch current overs count
	var currentOvers float64
	err := tx.Get(&currentOvers, `
		SELECT COALESCE((SELECT overs_bowled FROM player_match_stats WHERE innings_id = $1 AND player_id = $2), 0)`,
		inningsID, req.BowlerID)
	if err != nil {
		return err
	}

	newOvers := utils.CalculateNewOvers(currentOvers, isLegal, 1)

	query := `
		INSERT INTO player_match_stats (match_id, innings_id, player_id, runs_given, wickets_taken, maiden_overs, overs_bowled)
		VALUES ((SELECT match_id FROM innings WHERE id = $1), $1, $2, $3, $4, 0, $5)
		ON CONFLICT (innings_id, player_id) DO UPDATE SET
			runs_given    = player_match_stats.runs_given    + EXCLUDED.runs_given,
			wickets_taken = player_match_stats.wickets_taken + EXCLUDED.wickets_taken,
			overs_bowled  = $5,
			updated_at    = NOW()`

	_, err = tx.Exec(query, inningsID, req.BowlerID, bowlerRuns, bowlerWickets, newOvers)
	return err
}

// UpdateDismissedPlayerStatsTx updates the dismissed player
func UpdateDismissedPlayerStatsTx(tx *sqlx.Tx, inningsID string, req models.RecordBallRequest) error {
	if req.DismissedPlayerID == nil {
		return nil
	}

	isNotOut := false
	if req.DismissalType != nil && strings.ToLower(*req.DismissalType) == "retired_hurt" {
		isNotOut = true
	}

	var bowlerIDVal *string
	if req.DismissalType != nil && *req.DismissalType != "runout" && *req.DismissalType != "retired_out" && *req.DismissalType != "retired_hurt" {
		bowlerIDVal = &req.BowlerID
	}

	var fielderIDVal *string
	if req.FielderID != nil && *req.FielderID != "" {
		fielderIDVal = req.FielderID
	}

	_, err := tx.Exec(`
		UPDATE player_match_stats SET
			is_not_out     = $3,
			dismissal_type = $4::dismissal_type_enum,
			dismissed_by   = $5,
			fielder_id     = $6,
			updated_at     = NOW()
		WHERE innings_id = $1 AND player_id = $2`,
		inningsID, *req.DismissedPlayerID, isNotOut, req.DismissalType, bowlerIDVal, fielderIDVal,
	)
	return err
}

// FetchPlayerMatchStatsSummaryTx fetches statistics this is assocciated record bowl funtion 
func FetchPlayerMatchStatsSummaryTx(tx *sqlx.Tx, inningsID string, playerID string) (*models.PlayerStatsSummary, error) {
	var stats models.PlayerStatsSummary
	err := tx.Get(&stats, `
		SELECT player_id, runs_scored, balls_faced, fours, sixes, COALESCE(is_not_out, FALSE) as is_not_out, dismissal_type, dismissed_by, runs_given, wickets_taken, overs_bowled
		FROM player_match_stats
		WHERE innings_id = $1 AND player_id = $2`,
		inningsID, playerID,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

//splits them into batting and bowling teams inc common (when umpire open the scoring panel)
func FetchInningsPlayers(inningsID string) (*models.InningsPlayersDetails, error) {
	var innings struct {
		MatchID            string  `db:"match_id"`
		InningsNumber      int     `db:"innings_number"`
		BattingTeamID      string  `db:"batting_team_id"`
		BowlingTeamID      string  `db:"bowling_team_id"`
		BattingTeamName    string  `db:"batting_team_name"`
		BowlingTeamName    string  `db:"bowling_team_name"`
		MatchStatus        string  `db:"match_status"`
		InningsStatus      string  `db:"innings_status"`
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
			i.status AS innings_status,
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

	var strikerStats *models.PlayerStatsSummary
	var nonStrikerStats *models.PlayerStatsSummary

	if innings.ActiveStrikerID != nil && *innings.ActiveStrikerID != "" {
		var stats models.PlayerStatsSummary
		err := db.KhiladiDb.Get(&stats, `
			SELECT player_id, runs_scored, balls_faced, fours, sixes, COALESCE(is_not_out, FALSE) as is_not_out, dismissal_type, dismissed_by, runs_given, wickets_taken, overs_bowled
			FROM player_match_stats
			WHERE innings_id = $1 AND player_id = $2`,
			inningsID, *innings.ActiveStrikerID,
		)
		if err == nil {
			strikerStats = &stats
		} else {
			strikerStats = &models.PlayerStatsSummary{PlayerID: *innings.ActiveStrikerID, IsNotOut: true}
		}
	}
	if innings.ActiveNonStrikerID != nil && *innings.ActiveNonStrikerID != "" {
		var stats models.PlayerStatsSummary
		err := db.KhiladiDb.Get(&stats, `
			SELECT player_id, runs_scored, balls_faced, fours, sixes, COALESCE(is_not_out, FALSE) as is_not_out, dismissal_type, dismissed_by, runs_given, wickets_taken, overs_bowled
			FROM player_match_stats
			WHERE innings_id = $1 AND player_id = $2`,
			inningsID, *innings.ActiveNonStrikerID,
		)
		if err == nil {
			nonStrikerStats = &stats
		} else {
			nonStrikerStats = &models.PlayerStatsSummary{PlayerID: *innings.ActiveNonStrikerID, IsNotOut: true}
		}
	}

	var targetScore *int
	var firstInningsRuns *int
	var firstInningsWickets *int

	if innings.InningsNumber == 2 {
		var firstInnings struct {
			Runs    int `db:"total_runs"`
			Wickets int `db:"total_wickets"`
		}
		err := db.KhiladiDb.Get(&firstInnings, `SELECT total_runs, total_wickets FROM innings WHERE match_id = $1 AND innings_number = 1`, innings.MatchID)
		if err == nil {
			target := firstInnings.Runs + 1
			targetScore = &target
			firstInningsRuns = &firstInnings.Runs
			firstInningsWickets = &firstInnings.Wickets
		}
	}

	var playerStats []struct {
		PlayerID      string  `db:"player_id"`
		DismissalType *string `db:"dismissal_type"`
	}
	_ = db.KhiladiDb.Select(&playerStats, `
		SELECT player_id, dismissal_type 
		FROM player_match_stats 
		WHERE innings_id = $1`, inningsID)
	
	dismissedPlayerIDs := []string{}
	retiredHurtPlayerIDs := []string{}
	for _, ps := range playerStats {
		if ps.DismissalType != nil {
			if *ps.DismissalType == "retired_hurt" {
				retiredHurtPlayerIDs = append(retiredHurtPlayerIDs, ps.PlayerID)
			} else {
				dismissedPlayerIDs = append(dismissedPlayerIDs, ps.PlayerID)
			}
		}
	}

	details := &models.InningsPlayersDetails{
		MatchID:            innings.MatchID,
		InningsNumber:      innings.InningsNumber,
		MatchStatus:        innings.MatchStatus,
		InningsStatus:      innings.InningsStatus,
		BattingTeamID:      innings.BattingTeamID,
		BowlingTeamID:      innings.BowlingTeamID,
		BattingTeamName:    innings.BattingTeamName,
		BowlingTeamName:    innings.BowlingTeamName,
		BattingPlayers:     battingPlayers,
		BowlingPlayers:     bowlingPlayers,
		ActiveStrikerID:    innings.ActiveStrikerID,
		ActiveNonStrikerID: innings.ActiveNonStrikerID,
		ActiveBowlerID:     innings.ActiveBowlerID,
		StrikerStats:       strikerStats,
		NonStrikerStats:    nonStrikerStats,
		TotalRuns:          innings.TotalRuns,
		TotalWickets:       innings.TotalWickets,
		TotalOvers:         innings.TotalOvers,
		TotalOversLimit:    innings.TotalOversLimit,
		TossWinnerTeamID:   innings.TossWinnerTeamID,
		TossDecision:       innings.TossDecision,
		TargetScore:        targetScore,
		FirstInningsRuns:   firstInningsRuns,
		FirstInningsWickets: firstInningsWickets,
		BowlerStats:        bowlStats,
		DismissedPlayerIDs:   dismissedPlayerIDs,
		RetiredHurtPlayerIDs: retiredHurtPlayerIDs,
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
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, strikerID)
	}
	if nonStrikerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, nonStrikerID)
	}

	if bowlerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (innings_id, player_id) DO NOTHING`, matchID, inningsID, bowlerID)
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
			UPDATE player_match_stats 
			SET dismissal_type = NULL 
			WHERE innings_id = $1 AND player_id = $2 AND dismissal_type = 'retired_hurt'`,
			inningsID, *strikerID,
		)
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (innings_id, player_id) DO NOTHING`,
			matchID, inningsID, *strikerID,
		)
	}

	if nonStrikerID != nil && *nonStrikerID != "" {
		_, _ = tx.Exec(`
			UPDATE player_match_stats 
			SET dismissal_type = NULL 
			WHERE innings_id = $1 AND player_id = $2 AND dismissal_type = 'retired_hurt'`,
			inningsID, *nonStrikerID,
		)
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (innings_id, player_id) DO NOTHING`,
			matchID, inningsID, *nonStrikerID,
		)
	}

	if bowlerID != nil && *bowlerID != "" {
		_, _ = tx.Exec(`
			INSERT INTO player_match_stats (match_id, innings_id, player_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (innings_id, player_id) DO NOTHING`,
			matchID, inningsID, *bowlerID,
		)
	}

	return tx.Commit()
}





// inning refinement after inning completion and checks
func CheckAndUpdateInningsMatchCompletion(
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

	switch inningsNumber {
	case 1:
		if isOversFinished || isAllOut {
			isInningsComplete = true
		}
	case 2:
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
		// Finalize surviving batsmen (only those with no dismissal recorded)
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
				career_ducks = ps.career_ducks + CASE WHEN mb.total_runs = 0 AND NOT mb.is_not_out AND mb.total_balls_faced > 0 THEN 1 ELSE 0 END,
				career_fifties = ps.career_fifties + CASE WHEN mb.total_runs >= 50 AND mb.total_runs < 100 THEN 1 ELSE 0 END,
				career_hundreds = ps.career_hundreds + CASE WHEN mb.total_runs >= 100 THEN 1 ELSE 0 END,
				career_highest_score = GREATEST(ps.career_highest_score, COALESCE(mb.total_runs, 0)),
				career_maidens = ps.career_maidens + COALESCE(mbs.total_maidens, 0),
				career_highest_wickets = GREATEST(ps.career_highest_wickets, COALESCE(mbs.total_wickets, 0)),
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