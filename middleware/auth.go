package middleware

import (
	"fmt"
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		//split bearer and token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			return
		}

		tokenStr := parts[1]
		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Extract session_id from token
		sessionID := claims.SessionID
		if sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid session in token",
			})
			return
		}

		// Validate session from DB
		userID, err := dbhelper.GetUserIDBySession(sessionID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired session",
			})
			return
		}

		// Cross-check token with DB
		if userID != claims.UserID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token mismatch",
			})
			return
		}

		user, err := dbhelper.GetUserByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "user not found",
			})
			return
		}
		fmt.Println("in authMidd", userID)
		c.Set("session_id", sessionID)
		c.Set("user", user.ID)

		c.Next()
	}
}
