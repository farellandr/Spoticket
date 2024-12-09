package handlers

import (
	"fmt"
	"net/http"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type EditProfileRequest struct {
	Name        string `json:"name" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required,min=10,max=13"`
}

func GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Preload("Purchases").Preload("Payments").Preload("Events").Preload("Coupons").Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving user.")
		return
	}

	c.JSON(http.StatusOK, user)
}

func ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Where("id = ?", userID).First(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusUnauthorized, "You have no permission to change this user's password.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		helpers.RespondWithError(c, http.StatusUnauthorized, "Invalid credentials.")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to hash the password.")
		return
	}

	user.Password = string(hashedPassword)

	if err := gormDB.Save(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update password.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password updated successfully.",
	})
}

func EditProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}

	name := c.PostForm("name")
	phoneNumber := c.PostForm("phone_number")

	if name == "" || phoneNumber == "" {
		helpers.RespondWithError(c, http.StatusBadRequest, "Missing required fields.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Where("id = ?", userID).First(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusUnauthorized, "You have no permission to change this user's password.")
		return
	}

	user.Name = name
	user.PhoneNumber = phoneNumber

	profilePicture, err := c.FormFile("profile_picture")
	if err == nil {
		pfpPath, err := helpers.UploadFile(c, profilePicture, "profile_pictures")

		if user.ProfilePicture != nil {
			if err := helpers.DeleteFile(*user.ProfilePicture); err != nil {
				fmt.Printf("Error deleting old profile picture: %v\n", err)
			}
		}
		if err != nil {
			helpers.RespondWithError(c, http.StatusBadRequest, err.Error())
			return
		}
		user.ProfilePicture = &pfpPath
	}

	if err := gormDB.Save(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update password.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully.",
	})
}

func StreamProfilePicture(c *gin.Context) {
	userID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Where("id = ?", userID).First(&user).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving user.")
		return
	}

	if user.ProfilePicture == nil {
		helpers.RespondWithError(c, http.StatusNotFound, "No profile picture found.")
		return
	}

	c.File(*user.ProfilePicture)
}

func RemoveProfilePicture(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Where("id = ?", userID).First(&user).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving user.")
		return
	}

	if user.ProfilePicture != nil {
		if err := helpers.DeleteFile(*user.ProfilePicture); err != nil {
			fmt.Printf("Error deleting old profile picture: %v\n", err)
		}
	}

	user.ProfilePicture = nil

	if err := gormDB.Save(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update password.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile picture removed successfully.",
	})
}
