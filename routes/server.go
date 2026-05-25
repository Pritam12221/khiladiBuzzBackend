package routes

import (
	"khiladiBuzz/handlers"
	"khiladiBuzz/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ServerRoutes() *gin.Engine {

	r := gin.Default()

	serverCheck := r.Group("/v1")
	{
		serverCheck.POST("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "server chalrha",
			})
		})
	}

	//user routes
	userRoutes := r.Group("/v1")
	{
		userRoutes.POST("/login", handlers.LoginUser)
		userRoutes.POST("/register", handlers.RegisterUser)
		userRoutes.POST("/send-otp", handlers.SendOTP)
		userRoutes.POST("/forgot-password", handlers.ForgotPassword)
	}

	authRoutes:=r.Group("/v1")
	authRoutes.Use(middleware.AuthMiddleware())
	{
			authRoutes.POST("/logout",handlers.LogOutUser)
			authRoutes.POST("/teams", handlers.CreateTeam)
			authRoutes.GET("/fetchteams", handlers.FetchTeams)
			authRoutes.GET("/teams/:id/players", handlers.GetTeamPlayers)
			authRoutes.GET("/profile", handlers.GetProfile)
			authRoutes.PUT("/updateprofile", handlers.UpdateProfile)
			authRoutes.POST("/matches", handlers.CreateMatch)
	}

	return r;
}