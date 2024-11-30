package handlers

import (
	"net/http"
	"time"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	Province    string    `json:"province" binding:"required"`
	City        string    `json:"city" binding:"required"`
	District    string    `json:"district" binding:"required"`
	SubDistrict string    `json:"sub_district" binding:"required"`
	Location    string    `json:"location" binding:"required"`
}

func CreateEvent(c *gin.Context) {
	var req EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
		return
	}

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
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error checking user.")
		return
	}

	event := models.Event{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Province:    req.Province,
		City:        req.City,
		District:    req.District,
		SubDistrict: req.SubDistrict,
		Location:    req.Location,
		User:        user,
		UserID:      user.ID,
	}

	if err := gormDB.Create(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create event.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Event created successfully."})
}
