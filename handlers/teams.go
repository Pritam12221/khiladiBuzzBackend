package handlers

import (
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateTeam(c *gin.Context) {

	var req models.CreateTeamRequest

	// bind request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	
	captain, err := dbhelper.FindOrCreatePlayer(
		req.CaptainName,
		req.CaptainNumber,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get captain player",
		})
		return
	}

	
	userID := c.GetString("user_id")

	// create team
	teamID, err := dbhelper.CreateTeam(
		req.TeamName,
		captain.ID,
		userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create team",
		})
		return
	}

	// add captain into team_players
	err = dbhelper.AddPlayerToTeam(
		teamID,
		captain.ID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to add captain to team",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "team created successfully",
		"team_id": teamID,
	})
}