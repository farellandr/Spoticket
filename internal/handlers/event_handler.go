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
		UserID:      user.ID,
	}

	if err := gormDB.Create(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create event.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Event created successfully.",
		"event_id": event.ID,
	})
}

func GetEvent(c *gin.Context) {
	eventID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var event models.Event
	if err := gormDB.Preload("User").Preload("Tickets.Purchases").Where("id = ?", eventID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "Event not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving event.")
		return
	}

	c.JSON(http.StatusOK, event)
}

func ListEvents(c *gin.Context) {
	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	province := c.Query("province")
	city := c.Query("city")
	district := c.Query("district")

	pageNum, err := helpers.StringToInt(page)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid page number.")
		return
	}

	limitNum, err := helpers.StringToInt(limit)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid limit.")
		return
	}

	query := gormDB.Model(&models.Event{})
	if province != "" {
		query = query.Where("province = ?", province)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}
	if district != "" {
		query = query.Where("district = ?", district)
	}

	var totalCount int64
	query.Count(&totalCount)

	var events []models.Event
	offset := (pageNum - 1) * limitNum
	err = query.Preload("User").Preload("Tickets").Offset(offset).Limit(limitNum).Order("created_at DESC").Find(&events).Error
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving events.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total":       totalCount,
		"page":        pageNum,
		"limit":       limitNum,
		"total_pages": (totalCount + int64(limitNum) - 1) / int64(limitNum),
	})
}

func UpdateEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}

	var req EventRequest
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

	var event models.Event
	if err := gormDB.Where("id = ? AND user_id = ?", eventID, userID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusForbidden, "Event not found or you don't have permission to update.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error finding event.")
		return
	}

	event.Title = req.Title
	event.Description = req.Description
	event.StartTime = req.StartTime
	event.EndTime = req.EndTime
	event.Province = req.Province
	event.City = req.City
	event.District = req.District
	event.SubDistrict = req.SubDistrict
	event.Location = req.Location

	if err := gormDB.Save(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update event.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Event updated successfully.",
		"event":   event,
	})
}

func DeleteEvent(c *gin.Context) {
	eventID := c.Param("id")
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

	result := gormDB.Where("id = ? AND user_id = ?", eventID, userID).Delete(&models.Event{})
	if result.Error != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to delete event.")
		return
	}

	if result.RowsAffected == 0 {
		helpers.RespondWithError(c, http.StatusForbidden, "Event not found or you don't have permission to delete.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Event deleted successfully.",
	})
}
