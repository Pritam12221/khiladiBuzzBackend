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

	matchID, inningsID, err := dbhelper.CreateMatch(req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "match created successfully",
		"match_id":   matchID,
		"innings_id": inningsID,
	})
}


func RecordBall(c *gin.Context) {
	matchID := c.Param("id")
	inningsID := c.Param("innings_id")
	if matchID == "" || inningsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match id and innings id are required"})
		return
	}

	var req models.RecordBallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// extra type check
	validExtras := map[string]bool{"wide": true, "no_ball": true, "bye": true, "leg_bye": true}
	if req.ExtraType != nil && !validExtras[*req.ExtraType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid extra_type"})
		return
	}

	// type of out
	validDismissals := map[string]bool{
		"bowled": true, "caught": true, "lbw": true,
		"runout": true, "stumped": true, "hit_wicket": true, "retired_hurt": true,
	}
	if req.IsWicket && (req.DismissalType == nil || !validDismissals[*req.DismissalType]) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid dismissal_type required when is_wicket is true"})
		return
	}

	stats, err := dbhelper.RecordBall(inningsID, matchID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "ball recorded",
		"details": stats,
	})
}

