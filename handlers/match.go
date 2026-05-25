package handlers

import (
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateMatch(c *gin.Context) {
	var req models.CreateMatchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userID := c.GetString("user")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	matchID, err := dbhelper.CreateMatch(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "match created successfully",
		"match_id": matchID,
	})
}
