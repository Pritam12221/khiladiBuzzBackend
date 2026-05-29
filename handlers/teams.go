package handlers

import (
	"fmt"
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateTeam(c *gin.Context) {

	var req models.CreateTeamRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invali request",
		})
		return
	}

	userID := c.GetString("user")

	teamID, err := dbhelper.CreateTeamWithPlayers(
		req,
		userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create team",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "team created successfully",
		"team_id": teamID,
	})
}

func FetchTeams(c *gin.Context) {
	userID := c.GetString("user")

	teams, err := dbhelper.FetchTeams(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fecth teams"})
		return
	}

	c.JSON(http.StatusOK, teams)
}

func GetTeamPlayers(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	players, err := dbhelper.FetchTeamPlayers(teamID)
	fmt.Print(players)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error while fetching team players"})
		return
	}

	c.JSON(http.StatusOK, players)
}

func AddPlayerToTeam(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id is required"})
		return
	}

	var req models.CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player request"})
		return
	}

	player, err := dbhelper.FindOrCreatePlayerForTeam(req.PlayerName, req.PhoneNumber, req.Role, teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add player to team"})
		return
	}

	c.JSON(http.StatusCreated, player)
}


