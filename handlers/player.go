package handlers

import (
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"khiladiBuzz/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	userID := c.GetString("user")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	player, err := dbhelper.GetPlayerByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed fetch player profile"})
		return
	}

	c.JSON(http.StatusOK, player)
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := dbhelper.UpdatePlayerProfile(userID, req.PlayerName, req.PhoneNumber, req.Role, req.BattingStyle, req.BowlingStyle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update player profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}


func SearchPlayers(c *gin.Context) {
	userID := c.GetString("user")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		c.JSON(http.StatusOK, gin.H{"players": []interface{}{}})
		return
	}

	players, err := dbhelper.SearchPlayers(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search players"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"players": players})
}

func FetchAllPlayers(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))

	var limit, offset int
	if c.Query("page") == "" && c.Query("limit") == "" {
		limit = 1000
		offset = 0
	} else {
		limit, offset = utils.SetPagination(c)
	}

	players, err := dbhelper.GetAllPlayers(search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch all players"})
		return
	}

	c.JSON(http.StatusOK, players)
}

