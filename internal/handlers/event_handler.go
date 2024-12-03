package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateEvent(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")

	startTimeStr := c.PostForm("start_time")
	endTimeStr := c.PostForm("end_time")
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid start time format.")
		return
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid end time format.")
		return
	}

	province := c.PostForm("province")
	city := c.PostForm("city")
	district := c.PostForm("district")
	subDistrict := c.PostForm("sub_district")
	location := c.PostForm("location")

	var categories []string
	for i := 0; ; i++ {
		category := c.PostForm(fmt.Sprintf("categories[%d]", i))
		if category == "" {
			break
		}
		categories = append(categories, category)
	}

	if title == "" || description == "" || len(categories) == 0 {
		helpers.RespondWithError(c, http.StatusBadRequest, "Missing required fields.")
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
		helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
		return
	}

	var eventCategories []models.Category
	for _, categoryName := range categories {
		var category models.Category
		if err := gormDB.Where("name = ?", categoryName).FirstOrCreate(&category, models.Category{Name: categoryName}).Error; err != nil {
			helpers.RespondWithError(c, http.StatusInternalServerError, "Error processing categories.")
			return
		}
		eventCategories = append(eventCategories, category)
	}

	event := models.Event{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Province:    province,
		City:        city,
		District:    district,
		SubDistrict: subDistrict,
		Location:    location,
		UserID:      user.ID,
		Categories:  eventCategories,
	}

	bannerFile, err := c.FormFile("banner")
	if err == nil {
		bannerPath, err := helpers.UploadFile(c, bannerFile, "event_banners")
		if err != nil {
			helpers.RespondWithError(c, http.StatusBadRequest, err.Error())
			return
		}
		event.BannerPath = bannerPath
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
	if err := gormDB.Preload("Categories").Preload("User").Preload("Tickets.Purchases").Where("id = ?", eventID).First(&event).Error; err != nil {
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
	err = query.Preload("Categories").Preload("User").Preload("Tickets").Offset(offset).Limit(limitNum).Order("created_at DESC").Find(&events).Error
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

	title := c.PostForm("title")
	description := c.PostForm("description")

	startTimeStr := c.PostForm("start_time")
	endTimeStr := c.PostForm("end_time")
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid start time format.")
		return
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid end time format.")
		return
	}

	province := c.PostForm("province")
	city := c.PostForm("city")
	district := c.PostForm("district")
	subDistrict := c.PostForm("sub_district")
	location := c.PostForm("location")

	var categories []string
	for i := 0; ; i++ {
		category := c.PostForm(fmt.Sprintf("categories[%d]", i))
		if category == "" {
			break
		}
		categories = append(categories, category)
	}

	if title == "" || description == "" || len(categories) == 0 {
		helpers.RespondWithError(c, http.StatusBadRequest, "Missing required fields.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var event models.Event
	if err := gormDB.Preload("User").Where("id = ? AND user_id = ?", eventID, userID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusForbidden, "Event not found or you don't have permission to update.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error finding event.")
		return
	}

	event.Title = title
	event.Description = description
	event.StartTime = startTime
	event.EndTime = endTime
	event.Province = province
	event.City = city
	event.District = district
	event.SubDistrict = subDistrict
	event.Location = location

	bannerFile, err := c.FormFile("banner")
	if err == nil {
		bannerPath, err := helpers.UploadFile(c, bannerFile, "event_banners")

		if err := helpers.DeleteFile(event.BannerPath); err != nil {
			fmt.Printf("Error deleting old banner: %v\n", err)
		}
		if err != nil {
			helpers.RespondWithError(c, http.StatusBadRequest, err.Error())
			return
		}
		event.BannerPath = bannerPath
	}

	var updatedCategories []models.Category
	for _, categoryName := range categories {
		var category models.Category
		if err := gormDB.Where("name = ?", categoryName).FirstOrCreate(&category, models.Category{Name: categoryName}).Error; err != nil {
			helpers.RespondWithError(c, http.StatusInternalServerError, "Error processing categories.")
			return
		}
		updatedCategories = append(updatedCategories, category)
	}

	if err := gormDB.Save(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update event.")
		return
	}

	if err := gormDB.Model(&event).Association("Categories").Replace(updatedCategories); err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error updating categories.")
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
