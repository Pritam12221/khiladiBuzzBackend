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

	// Public routes 
	publicRoutes := r.Group("/v1")
	{
		publicRoutes.GET("/matches", handlers.FetchAllMatches)
		publicRoutes.GET("/matches/:id/scorecard", handlers.GetMatchScorecard)
		publicRoutes.GET("/players", handlers.FetchAllPlayers)
	}

	authRoutes:=r.Group("/v1")
	authRoutes.Use(middleware.AuthMiddleware())
	{
		
		authRoutes.POST("/logout", handlers.LogOutUser)
		authRoutes.GET("/profile", handlers.GetProfile)
		authRoutes.PUT("/updateprofile", handlers.UpdateProfile)
		authRoutes.GET("/fetchteams", handlers.FetchTeams)

		// Teams 
		teams := authRoutes.Group("/teams")
		{
			teams.POST("", handlers.CreateTeam)
			teams.GET("/:id/players", handlers.GetTeamPlayers)
			teams.POST("/:id/players", handlers.AddPlayerToTeam)
		}

		// Matches 
		matches := authRoutes.Group("/matches")
		{
			matches.POST("", handlers.CreateMatch)
			matches.POST("/:id/innings", handlers.CreateInnings)
			matches.POST("/:id/innings/:innings_id/ball", handlers.RecordBall)
		}

		// Innings 
		innings := authRoutes.Group("/innings")
		{
			innings.GET("/:id/players", handlers.GetInningsPlayers)
			innings.PUT("/:id/active-players", handlers.UpdateActivePlayers)
			innings.POST("/:id/undo", handlers.UndoLastBall)
			innings.POST("/:id/retire-hurt", handlers.RetireHurt)
		}

		// Players 
		players := authRoutes.Group("/players")
		{
			players.GET("/search", handlers.SearchPlayers)
		}
	}

	return r;
}