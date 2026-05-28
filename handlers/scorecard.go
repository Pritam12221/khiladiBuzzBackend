package handlers

import (
	"khiladiBuzz/database/dbhelper"
	"net/http"

	"github.com/gin-gonic/gin"
)

// get full match scorecard
func GetMatchScorecard(c *gin.Context) {
	matchID := c.Param("id")
	if matchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "match id is required"})
		return
	}

	scorecard, err := dbhelper.FetchMatchScorecard(matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scorecard)
}
