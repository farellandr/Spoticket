package handlers

import (
	"net/http"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRequest struct {
	Name string `json:"name" binding:"required,min=2"`
}

func CreateCategory(c *gin.Context) {
	var req CategoryRequest
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

	category := models.Category{
		ID:   uuid.New(),
		Name: req.Name,
	}

	if err := gormDB.Create(&category).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create category.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Category created successfully.",
		"event_id": category.ID,
	})
}

func ListCategories(c *gin.Context) {
	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

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

	query := gormDB.Model(&models.Category{})
	var totalCount int64
	query.Count(&totalCount)

	var categories []models.Category
	offset := (pageNum - 1) * limitNum
	err = query.Offset(offset).Limit(limitNum).Order("created_at DESC").Find(&categories).Error
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving categories.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories":  categories,
		"total":       totalCount,
		"page":        pageNum,
		"limit":       limitNum,
		"total_pages": (totalCount + int64(limitNum) - 1) / int64(limitNum),
	})
}

func UpdateCategory(c *gin.Context) {
	categoryID := c.Param("id")

	var req CategoryRequest
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

	var category models.Category
	if err := gormDB.Where("id = ?", categoryID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusForbidden, "Category not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error finding category.")
		return
	}

	category.Name = req.Name

	if err := gormDB.Save(&category).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update category.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Category updated successfully.",
		"category": category,
	})
}

func DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	result := gormDB.Where("id = ?", categoryID).Delete(&models.Category{})
	if result.Error != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to delete category.")
		return
	}

	if result.RowsAffected == 0 {
		helpers.RespondWithError(c, http.StatusForbidden, "Category not found or you don't have permission to delete.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Category deleted successfully.",
	})
}
