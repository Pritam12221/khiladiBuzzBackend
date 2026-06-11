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

	details, err := dbhelper.FetchInningsPlayers(inningsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch innings players"})
		return
	}

	c.JSON(http.StatusOK, details)
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

func UndoLastBall(c *gin.Context) {
	inningsID := c.Param("id")
	if inningsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "innings id is required"})
		return
	}

	details, err := dbhelper.UndoLastBall(inningsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}


