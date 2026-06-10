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
		c.JSON(http.StatusBadRequest, gin.H{"error": "login failed"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registration request"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	// Create user
	user, err := dbhelper.CreateUser(req.Name, req.PhoneNumber, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}



	sessionID, err := dbhelper.CreateUserSession(*user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return	
	}

	//generate a token
	token, err := utils.GenerateToken(*user.UserID, sessionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"token":token,
	})
}



func LogOutUser(c* gin.Context){
		sessionId:=c.GetString("session_id");

		if sessionId== ""{
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invlid session"})
		}

		err := dbhelper.DeleteUserSession(sessionId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

func SendOTP(c *gin.Context) {
	var req models.SendOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":"Please enter a valid 10-digit mobile number.",
		})
		return
	}

	exists, err := dbhelper.IsUserExist(req.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":"Unable to verify your number right now. Please try again.",
		})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":"No account found with this mobile number. Please register first.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

func ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":"Please fill all fields correctly.Password must be at least 6 characters and OTP must be 4 digits.",
		})
		return
	}

	if req.OTP != "8080" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "The OTP you entered is incorrect. Please check and try again.",
		})
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process your password. Please try again.",
		})
		return
	}

	err = dbhelper.UpdateUserPassword(req.PhoneNumber, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not reset your password. Please try again later.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
