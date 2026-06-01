package dbhelper

import (
	"fmt"
	db "khiladiBuzz/database"
	"khiladiBuzz/models"
	"math"
)

// populateInningsData fetches, aggregates, and formats all batting, bowling, and extras data for a single
func populateInningsData(match *matchRow, inn inningsRow) (models.InningsData, error) {

		teamName := match.TeamA
	if inn.BattingTeamID == match.TeamBID {
		teamName = match.TeamB
	}

	innData := models.InningsData{
		TeamName:      teamName,
		Runs:          inn.TotalRuns,
		Wickets:       inn.TotalWickets,
		Overs:         fmt.Sprintf("%.1f", inn.TotalOvers),
		Batting:       []models.BatsmanRow{},
		Bowling:       []models.BowlerRow{},
		YetToBat:      []string{},
		FallOfWickets: []string{}, 
		TopBatsmen:    []models.TopBatsman{},
		TopBowlers:    []models.TopBowler{},
		Extras:        models.ExtrasSummary{},
	}

	// track order of player which they are batting
	balls, _ := FetchInningsBalls(inn.ID)
	orderedBatsmenIDs, seenBatsmen := getOrderedBatsmen(balls, inn.ActiveStrikerID, inn.ActiveNonStrikerID)
	orderedBowlersIDs, seenBowlers := getOrderedBowlers(balls, inn.ActiveBowlerID)

	// Populate Batting stats
	batStats, err := FetchBattingStats(match.ID, inn.ID, inn.BattingTeamID)
	if err != nil {
		fmt.Printf("livescorecard: FetchBattingStats error: %v\n", err)
		return models.InningsData{}, err
	}
	innData.Batting = formatBattingStats(batStats, orderedBatsmenIDs, seenBatsmen, inn.ActiveStrikerID, inn.ActiveNonStrikerID)

	// Populate bowlers stats 
	activeBowlerID := ""
	if inn.ActiveBowlerID != nil {
		activeBowlerID = *inn.ActiveBowlerID
	}
	bowlStats, err := FetchBowlingStats(match.ID, inn.ID, inn.BowlingTeamID)
	if err != nil {
		fmt.Printf("livescorecard: FetchBowlingStats error: %v\n", err)
		return models.InningsData{}, err
	}
	
	if activeBowlerID != "" && inn.ActiveBowlerName != nil {
		found := false
		for _, bs := range bowlStats {
			if bs.PlayerID == activeBowlerID {
				found = true
				break
			}
		}
		if !found {
			bowlStats = append(bowlStats, models.BowlStat{
				PlayerID: activeBowlerID,
				Name:     *inn.ActiveBowlerName,
			})
		}
	}
	innData.Bowling = formatBowlingStats(bowlStats, orderedBowlersIDs, seenBowlers, activeBowlerID)

	// fetch total innings
	innData.Extras, _ = FetchInningsExtras(inn.ID)

	// Populate Yet to Bat Players
	innData.YetToBat, _ = FetchYetToBat(match.ID, inn.BattingTeamID, seenBatsmen)

	// summary table 
	if match.Status == "completed" {
		innData.TopBatsmen, _ = FetchTopBatsmenSummary(match.ID, inn.BattingTeamID)
		innData.TopBowlers, _ = FetchTopBowlersSummary(match.ID, inn.BowlingTeamID)
	}

	// to handle solo player
	normalizeInningsData(&innData)

	return innData, nil
}

// FetchMatchScorecard fetches all aggregated innings and match statistics for a public scorecard.
func FetchMatchScorecard(matchID string) (*models.MatchDetail, error) {

	// Fetch match-level data
	match, err := fetchMatchMetadata(matchID)
	if err != nil {
		return nil, err
	}

	scorecard := assembleMatchDetail(match)

	// Fetch all innings in the match
	innings, err := fetchMatchInningsRows(matchID)
	if err != nil {
		return nil, err
	}

	for _, inn := range innings {
		innData, err := populateInningsData(match, inn)
		if err != nil {
			return nil, err
		}

		if inn.InningsNumber == 1 {
			scorecard.Innings1 = innData
		} else {
			scorecard.Innings2 = &innData
		}
	}

	if match.Status == "completed" && match.WinnerTeamID == nil {
		if scorecard.Innings2 != nil && scorecard.Innings1.Runs == scorecard.Innings2.Runs {
			text := "Match tied"
			scorecard.ResultText = &text
		}
	}

	// Fetch Squads
	squad1, _ := FetchMatchSquad(matchID, match.TeamAID, match.TeamA)
	if squad1 != nil {
		scorecard.Squad1 = squad1
	}
	squad2, _ := FetchMatchSquad(matchID, match.TeamBID, match.TeamB)
	if squad2 != nil {
		scorecard.Squad2 = squad2
	}

	return scorecard, nil
}

func FetchMatchSquad(matchID, teamID, teamName string) (*models.PlayingSquad, error) {
	var players []models.SquadPlayer
	query := `
		SELECT DISTINCT p.id as player_id, u.name as player_name, p.role, p.batting_style, p.bowling_style, COALESCE(mp.is_captain, FALSE) as is_captain
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = $1 AND mp.team_id = $2
		WHERE (mp.match_id = $1 AND mp.team_id = $2)
		   OR (p.id = (SELECT common_player_id FROM matches WHERE id = $1))
	`
	err := db.KhiladiDb.Select(&players, query, matchID, teamID)
	if err != nil {
		return nil, err
	}

	return &models.PlayingSquad{
		TeamName:  teamName,
		Players:   players,
	}, nil
}



type matchRow struct {
	ID               string  `db:"id"`
	TeamA            string  `db:"team_a_name"`
	TeamB            string  `db:"team_b_name"`
	TeamAID          string  `db:"team1_id"`
	TeamBID          string  `db:"team2_id"`
	Status           string  `db:"status"`
	TotalOvers       int     `db:"total_overs"`
	TossWinnerTeamID string  `db:"toss_winner_team_id"`
	TossDecision     string  `db:"toss_decision"`
	CreatedAt        string  `db:"created_at"`
	WinnerTeamID     *string `db:"winner_team_id"`
	HostName         *string `db:"host_name"`
	UmpireName       *string `db:"umpire_name"`
}

func fetchMatchMetadata(matchID string) (*matchRow, error) {
	var match matchRow
	query := `
		SELECT 
			m.id, 
			t1.team_name as team_a_name, 
			t2.team_name as team_b_name,
			m.team1_id,
			m.team2_id,
			m.status, 
			m.total_overs, 
			m.toss_winner_team_id, 
			m.toss_decision,
			TO_CHAR(m.created_at, 'Mon DD, YYYY') as created_at,
			m.winner_team_id,
			u_host.name as host_name,
			u_umpire.name as umpire_name
		FROM matches m
		JOIN teams t1 ON m.team1_id = t1.id
		JOIN teams t2 ON m.team2_id = t2.id
		LEFT JOIN users u_host ON m.host_id = u_host.id
		LEFT JOIN users u_umpire ON m.umpire_id = u_umpire.id
		WHERE m.id = $1`
	err := db.KhiladiDb.Get(&match, query, matchID)
	return &match, err
}

type inningsRow struct {
	ID                 string  `db:"id"`
	InningsNumber      int     `db:"innings_number"`
	BattingTeamID      string  `db:"batting_team_id"`
	BowlingTeamID      string  `db:"bowling_team_id"`
	TotalRuns          int     `db:"total_runs"`
	TotalWickets       int     `db:"total_wickets"`
	TotalOvers         float64 `db:"total_overs"`
	ActiveStrikerID    *string `db:"active_striker_id"`
	ActiveNonStrikerID *string `db:"active_non_striker_id"`
	ActiveBowlerID     *string `db:"active_bowler_id"`
	ActiveBowlerName   *string `db:"active_bowler_name"`
}

func fetchMatchInningsRows(matchID string) ([]inningsRow, error) {
	var innings []inningsRow
	query := `
		SELECT 
			i.id, i.innings_number, i.batting_team_id, i.bowling_team_id,
			i.total_runs, i.total_wickets, i.total_overs,
			i.active_striker_id, i.active_non_striker_id, i.active_bowler_id,
			ub.name as active_bowler_name
		FROM innings i
		LEFT JOIN player_stats pb ON i.active_bowler_id = pb.id
		LEFT JOIN users ub ON pb.user_id = ub.id
		WHERE i.match_id = $1
		ORDER BY i.innings_number ASC`
	err := db.KhiladiDb.Select(&innings, query, matchID)
	return innings, err
}

// FetchInningsBalls retrieves all deliveries in chronological order for the innings
func FetchInningsBalls(inningsID string) ([]models.BallRow, error) {
	var balls []models.BallRow
	query := `
		SELECT striker_id, non_striker_id, bowler_id 
		FROM balls 
		WHERE innings_id = $1 
		ORDER BY over_number ASC, ball_number ASC, created_at ASC`
	err := db.KhiladiDb.Select(&balls, query, inningsID)
	return balls, err
}

// FetchBattingStats fetches raw batting match stats for all squad players
func FetchBattingStats(matchID, inningsID, battingTeamID string) ([]models.BatStat, error) {
	var batStats []models.BatStat
	query := `
		SELECT 
			p.id as player_id,
			u.name as player_name,
			COALESCE(pms.runs_scored, 0) as runs_scored,
			COALESCE(pms.balls_faced, 0) as balls_faced,
			COALESCE(pms.fours, 0) as fours,
			COALESCE(pms.sixes, 0) as sixes,
			COALESCE(pms.is_not_out, TRUE) as is_not_out,
			pms.dismissal_type,
			u_bowl.name as bowler_name,
			u_field.name as fielder_name
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = $1 AND mp.team_id = $2
		LEFT JOIN player_match_stats pms ON pms.player_id = p.id AND pms.match_id = $1
		LEFT JOIN player_stats pb ON pms.dismissed_by = pb.id
		LEFT JOIN users u_bowl ON pb.user_id = u_bowl.id
		LEFT JOIN player_stats pf ON pms.fielder_id = pf.id
		LEFT JOIN users u_field ON pf.user_id = u_field.id
		WHERE (mp.match_id = $1 AND mp.team_id = $2)
		   OR (p.id = (SELECT common_player_id FROM matches WHERE id = $1))`
	err := db.KhiladiDb.Select(&batStats, query, matchID, battingTeamID)
	return batStats, err
}

// FetchBowlingStats fetches raw bowling match stats for all squad players
func FetchBowlingStats(matchID, inningsID, bowlingTeamID string) ([]models.BowlStat, error) {
	var bowlStats []models.BowlStat
	query := `
		SELECT 
			p.id as player_id,
			u.name as player_name,
			COALESCE(pms.overs_bowled, 0.0) as overs_bowled,
			COALESCE(pms.runs_given, 0) as runs_given,
			COALESCE(pms.wickets_taken, 0) as wickets_taken
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = $1 AND mp.team_id = $2
		LEFT JOIN player_match_stats pms ON pms.player_id = p.id AND pms.match_id = $1
		WHERE (mp.match_id = $1 AND mp.team_id = $2)
		   OR (p.id = (SELECT common_player_id FROM matches WHERE id = $1))`
	err := db.KhiladiDb.Select(&bowlStats, query, matchID, bowlingTeamID)
	return bowlStats, err
}

// FetchInningsExtras calculates extras from the balls table
func FetchInningsExtras(inningsID string) (models.ExtrasSummary, error) {
	var extras models.ExtrasSummary
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN extra_type = 'wide' THEN extras_runs + 1 ELSE 0 END), 0) as wides,
			COALESCE(SUM(CASE WHEN extra_type = 'no_ball' THEN 1 ELSE 0 END), 0) as noBalls,
			COALESCE(SUM(CASE WHEN extra_type = 'bye' THEN extras_runs ELSE 0 END), 0) as byes,
			COALESCE(SUM(CASE WHEN extra_type = 'leg_bye' THEN extras_runs ELSE 0 END), 0) as legByes,
			COALESCE(SUM(extras_runs + CASE WHEN extra_type IN ('wide', 'no_ball') THEN 1 ELSE 0 END), 0) as total
		FROM balls 
		WHERE innings_id = $1`
	err := db.KhiladiDb.Get(&extras, query, inningsID)
	return extras, err
}

// FetchYetToBat retrieves all batting squad players who haven't faced a ball yet
func FetchYetToBat(matchID, battingTeamID string, seenBatsmen map[string]bool) ([]string, error) {
	var teamPlayers []models.PlayerNameRow
	query := `
		SELECT DISTINCT p.id as player_id, u.name as player_name
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = $1 AND mp.team_id = $2
		WHERE (mp.match_id = $1 AND mp.team_id = $2)
		   OR (p.id = (SELECT common_player_id FROM matches WHERE id = $1))`
	err := db.KhiladiDb.Select(&teamPlayers, query, matchID, battingTeamID)
	if err != nil {
		return nil, err
	}

	var yetToBat []string
	for _, tp := range teamPlayers {
		if !seenBatsmen[tp.PlayerID] {
			yetToBat = append(yetToBat, tp.Name)
		}
	}
	return yetToBat, nil
}

// FetchTopBatsmenSummary returns the top 2 batsmen directly sorted by runs scored
func FetchTopBatsmenSummary(matchID, battingTeamID string) ([]models.TopBatsman, error) {
	var rows []models.TopBatsmanRow
	query := `
		SELECT u.name as player_name, pms.runs_scored, pms.balls_faced
		FROM player_match_stats pms
		JOIN player_stats p ON pms.player_id = p.id
		JOIN users u ON p.user_id = u.id
		JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = pms.match_id
		WHERE pms.match_id = $1 AND mp.team_id = $2
		  AND (pms.runs_scored > 0 OR pms.balls_faced > 0)
		ORDER BY pms.runs_scored DESC, pms.balls_faced ASC
		LIMIT 2`
	err := db.KhiladiDb.Select(&rows, query, matchID, battingTeamID)
	if err != nil {
		return nil, err
	}

	var topBat []models.TopBatsman
	for _, r := range rows {
		topBat = append(topBat, models.TopBatsman{Name: r.Name, Score: fmt.Sprintf("%d (%d)", r.Runs, r.Balls)})
	}
	return topBat, nil
}

// FetchTopBowlersSummary returns the top 2 bowlers directly sorted by wickets taken
func FetchTopBowlersSummary(matchID, bowlingTeamID string) ([]models.TopBowler, error) {
	var rows []models.TopBowlerRow
	query := `
		SELECT u.name as player_name, pms.wickets_taken, pms.runs_given, pms.overs_bowled
		FROM player_match_stats pms
		JOIN player_stats p ON pms.player_id = p.id
		JOIN users u ON p.user_id = u.id
		JOIN match_players mp ON mp.player_id = p.id AND mp.match_id = pms.match_id
		WHERE pms.match_id = $1 AND mp.team_id = $2
		  AND pms.overs_bowled > 0
		ORDER BY pms.wickets_taken DESC, pms.runs_given ASC
		LIMIT 2`
	err := db.KhiladiDb.Select(&rows, query, matchID, bowlingTeamID)
	if err != nil {
		return nil, err
	}

	var topBowl []models.TopBowler
	for _, r := range rows {
		topBowl = append(topBowl, models.TopBowler{Name: r.Name, Figures: fmt.Sprintf("%d/%d (%.1f)", r.Wickets, r.Runs, r.Overs)})
	}
	return topBowl, nil
}



func getOrderedBatsmen(balls []models.BallRow, activeStriker, activeNonStriker *string) ([]string, map[string]bool) {
	var ordered []string
	seen := map[string]bool{}

	for _, b := range balls {
		if !seen[b.StrikerID] {
			seen[b.StrikerID] = true
			ordered = append(ordered, b.StrikerID)
		}
		if b.NonStrikerID != nil && *b.NonStrikerID != "" && !seen[*b.NonStrikerID] {
			seen[*b.NonStrikerID] = true
			ordered = append(ordered, *b.NonStrikerID)
		}
	}

	if activeStriker != nil && *activeStriker != "" && !seen[*activeStriker] {
		seen[*activeStriker] = true
		ordered = append(ordered, *activeStriker)
	}
	if activeNonStriker != nil && *activeNonStriker != "" && !seen[*activeNonStriker] {
		seen[*activeNonStriker] = true
		ordered = append(ordered, *activeNonStriker)
	}

	return ordered, seen
}

func getOrderedBowlers(balls []models.BallRow, activeBowler *string) ([]string, map[string]bool) {
	var ordered []string
	seen := map[string]bool{}

	for _, b := range balls {
		if !seen[b.BowlerID] {
			seen[b.BowlerID] = true
			ordered = append(ordered, b.BowlerID)
		}
	}

	if activeBowler != nil && *activeBowler != "" && !seen[*activeBowler] {
		seen[*activeBowler] = true
		ordered = append(ordered, *activeBowler)
	}

	return ordered, seen
}

func formatBattingStats(batStats []models.BatStat, orderedBatsmenIDs []string, seenBatsmen map[string]bool,activeStrikerID *string,
	activeNonStrikerID *string) []models.BatsmanRow {
	statsMap := make(map[string]models.BatStat)
	for _, b := range batStats {
		if seenBatsmen[b.PlayerID] || b.Balls > 0 || b.Runs > 0 {
			statsMap[b.PlayerID] = b
		}
	}

	var formatted []models.BatsmanRow
	for _, playerID := range orderedBatsmenIDs {
		if b, exists := statsMap[playerID]; exists {
			sr := "0.00"
			if b.Balls > 0 {
				sr = fmt.Sprintf("%.2f", float64(b.Runs)/float64(b.Balls)*100)
			}

			isActive := false
			isStriker := false
			if activeStrikerID != nil && *activeStrikerID == b.PlayerID {
				isActive = true
				isStriker = true
			}
			if activeNonStrikerID != nil && *activeNonStrikerID == b.PlayerID {
				isActive = true
			}

			formatted = append(formatted, models.BatsmanRow{
				Name:          b.Name,
				Runs:          b.Runs,
				Balls:         b.Balls,
				Fours:         b.Fours,
				Sixes:         b.Sixes,
				SR:            sr,
				IsActive:      isActive,
				IsStriker:     isStriker,
				IsNotOut:      b.IsNotOut,
				DismissalType: b.DismissalType,
				BowlerName:    b.DismissedBy,
				FielderName:   b.FielderName,
			})
		}
	}
	return formatted
}

func formatBowlingStats(bowlStats []models.BowlStat, orderedBowlersIDs []string, seenBowlers map[string]bool, activeBowlerID string) []models.BowlerRow {
	statsMap := make(map[string]models.BowlStat)
	for _, b := range bowlStats {
		if b.PlayerID == activeBowlerID || seenBowlers[b.PlayerID] {
			statsMap[b.PlayerID] = b
		}
	}

	var formatted []models.BowlerRow
	for _, playerID := range orderedBowlersIDs {
		if b, exists := statsMap[playerID]; exists {
			oversDec := b.OversBowled
			wholeOvers := math.Floor(oversDec)
			balls := int(math.Round((oversDec - wholeOvers) * 10))
			totalBalls := int(wholeOvers)*6 + balls

			econ := "0.00"
			if totalBalls > 0 {
				econ = fmt.Sprintf("%.2f", float64(b.RunsGiven)/float64(totalBalls)*6)
			}

			formatted = append(formatted, models.BowlerRow{
				Name:     b.Name,
				Overs:    fmt.Sprintf("%.1f", b.OversBowled),
				Runs:     b.RunsGiven,
				Wickets:  b.Wickets,
				Econ:     econ,
				Maidens:  0,
				IsActive: b.PlayerID == activeBowlerID,
			})
		}
	}
	return formatted
}

func assembleMatchDetail(match *matchRow) *models.MatchDetail {
	tossWinnerName := match.TeamA
	if match.TossWinnerTeamID == match.TeamBID {
		tossWinnerName = match.TeamB
	}

	statusText := ""
	switch match.Status {
	case "live":
		statusText = "LIVE"
	case "completed":
		statusText = "COMPLETED"
	}

	var resultTextVal *string
	if match.Status == "completed" {
		var text string
		if match.WinnerTeamID != nil {
			switch *match.WinnerTeamID {
			case match.TeamAID:
				text = match.TeamA + " won the match"
			case match.TeamBID:
				text = match.TeamB + " won the match"
			default:
				text = "Match tied"
			}
		} else {
			text = "Match completed"
		}
		resultTextVal = &text
	}

	umpireName := "Match Umpire"
	if match.UmpireName != nil {
		umpireName = *match.UmpireName
	}
	hostName := "KhiladiBuzz"
	if match.HostName != nil {
		hostName = *match.HostName
	}

	return &models.MatchDetail{
		ID:           match.ID,
		TeamA:        match.TeamA,
		TeamB:        match.TeamB,
		Status:       match.Status,
		StatusText:   statusText,
		ResultText:   resultTextVal,
		Date:         match.CreatedAt,
		TotalOvers:   match.TotalOvers,
		TossWinner:   tossWinnerName,
		TossDecision: match.TossDecision,
		Umpire:       umpireName,
		Host:         hostName,
	}
}

func normalizeInningsData(innData *models.InningsData) {
	if innData.Batting == nil {
		innData.Batting = []models.BatsmanRow{}
	}
	if innData.Bowling == nil {
		innData.Bowling = []models.BowlerRow{}
	}
	if innData.YetToBat == nil {
		innData.YetToBat = []string{}
	}
	if innData.FallOfWickets == nil {
		innData.FallOfWickets = []string{}
	}
	if innData.TopBatsmen == nil {
		innData.TopBatsmen = []models.TopBatsman{}
	}
	if innData.TopBowlers == nil {
		innData.TopBowlers = []models.TopBowler{}
	}
}
