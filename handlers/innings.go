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

	battingPlayers, bowlingPlayers, battingTeamName, bowlingTeamName, battingTeamID, bowlingTeamID, matchID, inningsNumber, matchStatus, activeStrikerID, activeNonStrikerID, activeBowlerID, totalRuns, totalWickets, totalOvers, totalOversLimit, tossWinnerTeamID, tossDecision, err := dbhelper.FetchInningsPlayers(inningsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch innings players"})
		return
	}

	var targetScore *int
	if inningsNumber == 2 {
		var firstInningsRuns int
		err := db.KhiladiDb.Get(&firstInningsRuns, `SELECT total_runs FROM innings WHERE match_id = $1 AND innings_number = 1`, matchID)
		if err == nil {
			target := firstInningsRuns + 1
			targetScore = &target
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"match_id":              matchID,
		"innings_number":        inningsNumber,
		"match_status":          matchStatus,
		"batting_team_id":       battingTeamID,
		"bowling_team_id":       bowlingTeamID,
		"batting_team_name":     battingTeamName,
		"bowling_team_name":     bowlingTeamName,
		"batting_players":       battingPlayers,
		"bowling_players":       bowlingPlayers,
		"active_striker_id":     activeStrikerID,
		"active_non_striker_id": activeNonStrikerID,
		"active_bowler_id":      activeBowlerID,
		"total_runs":            totalRuns,
		"total_wickets":         totalWickets,
		"total_overs":           totalOvers,
		"total_overs_limit":     totalOversLimit,
		"toss_winner_team_id":   tossWinnerTeamID,
		"toss_decision":         tossDecision,
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
