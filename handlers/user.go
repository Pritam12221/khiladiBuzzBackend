package handlers

import (
	"khiladiBuzz/database/dbhelper"
	"khiladiBuzz/models"
	"khiladiBuzz/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)


func LoginUser(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := dbhelper.GetUserByPhoneNumber(req.PhoneNumber, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

		sessionID, err := dbhelper.CreateUserSession(user.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	//generate a token
	token, err := utils.GenerateToken(user.ID,  sessionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token":token,
	})

}



func RegisterUser(c *gin.Context) {
	var req models.UserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//check  user exist or not
	isExists, _ := dbhelper.IsUserExist(req.PhoneNumber)
	if isExists {
		c.JSON(http.StatusConflict, gin.H{"error": "User alreaady exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user
	userID, err := dbhelper.CreateUser(req.Name, req.PhoneNumber, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}


	_, err = dbhelper.CreatePlayerForUser(req.Name, req.PhoneNumber, userID)
	if err != nil {
		c.Error(err)
	}

	sessionID, err := dbhelper.CreateUserSession(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return	
	}

	//generate a token
	token, err := utils.GenerateToken(userID,  sessionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": userID,
		"token":token,
	})
}



func LogOutUser(c* gin.Context){
		sessionId:=c.GetString("session_id");

		if sessionId== ""{
			c.JSON(http.StatusBadRequest, gin.H{"error": "Logout failed"})
		}

		err := dbhelper.DeleteUserSession(sessionId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}