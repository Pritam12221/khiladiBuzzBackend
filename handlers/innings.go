package handlers

import (
	db "khiladiBuzz/database"
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetInningsPlayers(c *gin.Context) {
	inningsID := c.Param("id")
	if inningsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "innings id is required"})
		return
	}

	details, err := dbhelper.FetchInningsPlayers(inningsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch innings players"})
		return
	}

	var targetScore *int
	if details.InningsNumber == 2 {
		var firstInningsRuns int
		err := db.KhiladiDb.Get(&firstInningsRuns, `SELECT total_runs FROM innings WHERE match_id = $1 AND innings_number = 1`, details.MatchID)
		if err == nil {
			target := firstInningsRuns + 1
			targetScore = &target
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"match_id":              details.MatchID,
		"innings_number":        details.InningsNumber,
		"match_status":          details.MatchStatus,
		"batting_team_id":       details.BattingTeamID,
		"bowling_team_id":       details.BowlingTeamID,
		"batting_team_name":     details.BattingTeamName,
		"bowling_team_name":     details.BowlingTeamName,
		"batting_players":       details.BattingPlayers,
		"bowling_players":       details.BowlingPlayers,
		"active_striker_id":     details.ActiveStrikerID,
		"active_non_striker_id": details.ActiveNonStrikerID,
		"active_bowler_id":      details.ActiveBowlerID,
		"total_runs":            details.TotalRuns,
		"total_wickets":         details.TotalWickets,
		"total_overs":           details.TotalOvers,
		"total_overs_limit":     details.TotalOversLimit,
		"toss_winner_team_id":   details.TossWinnerTeamID,
		"toss_decision":         details.TossDecision,
		"target_score":          targetScore,
	})
}

func UpdateActivePlayers(c *gin.Context) {
	inningsID := c.Param("id")
	if inningsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "innings id is required"})
		return
	}

	var req models.UpdateActivePlayersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	err := dbhelper.UpdateActivePlayers(inningsID, req.ActiveStrikerID, req.ActiveNonStrikerID, req.ActiveBowlerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update active players"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "active players updated successfully"})
}



func CreateInnings(c *gin.Context) {
	matchID := c.Param("id")
	if matchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match id is required"})
		return
	}

	var req models.CreateInningsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to create innings"})
		return
	}

	inningsID, err := dbhelper.CreateInnings(matchID, req.InningsNumber, req.BattingTeamID, req.BowlingTeamID, req.Status, req.StrikerID, req.NonStrikerID, req.BowlerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid request payload"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "innings created successfully",
		"innings_id": inningsID,
	})
}
