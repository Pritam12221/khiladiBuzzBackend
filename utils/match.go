package utils

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

func SetPagination(c *gin.Context) (int, int) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	return limit, offset

}

func CalculateNewOvers(currentOvers float64, isLegal bool, count int) float64 {
	if !isLegal {
		return currentOvers
	}
	completedOvers := int(currentOvers)
	balls := int(math.Round((currentOvers - float64(completedOvers)) * 10))

	totalBalls := completedOvers*6 + balls + count
	if totalBalls < 0 {
		totalBalls = 0
	}
	return float64(totalBalls/6) + float64(totalBalls%6)*0.1
}
