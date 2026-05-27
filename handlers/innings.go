package handlers

import (
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

	battingPlayers, bowlingPlayers, battingTeamName, bowlingTeamName, battingTeamID, bowlingTeamID, matchID, err := dbhelper.FetchInningsPlayers(inningsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"match_id":          matchID,
		"batting_team_id":   battingTeamID,
		"bowling_team_id":   bowlingTeamID,
		"batting_team_name": battingTeamName,
		"bowling_team_name": bowlingTeamName,
		"batting_players":   battingPlayers,
		"bowling_players":   bowlingPlayers,
	})
}



func CreateInnings(c *gin.Context) {
	matchID := c.Param("id")
	if matchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match id is required"})
		return
	}

	var req models.CreateInningsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inningsID, err := dbhelper.CreateInnings(matchID, req.InningsNumber, req.BattingTeamID, req.BowlingTeamID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "innings created successfully",
		"innings_id": inningsID,
	})
}
