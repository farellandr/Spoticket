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

type CouponRequest struct {
	Name        string    `json:"name" binding:"required"`
	Code        *string   `json:"code"`
	Limit       int       `json:"limit" binding:"required"`
	ValidAt     time.Time `json:"valid_at" binding:"required"`
	ExpiredAt   time.Time `json:"expired_at" binding:"required"`
	Description string    `json:"description" binding:"required"`
}

func CreateCoupon(c *gin.Context) {
	var req CouponRequest
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

	// var coupon models.Coupon
	// if err := gormDB.Where("id = ? AND user_id = ?", req.EventID, userID).First(&event).Error; err != nil {
	// 	if err == gorm.ErrRecordNotFound {
	// 		helpers.RespondWithError(c, http.StatusForbidden, "Event not found or you don't have permission to modify it.")
	// 		return
	// 	}
	// 	helpers.RespondWithError(c, http.StatusInternalServerError, "Error verifying event ownership.")
	// 	return
	// }

	coupon := models.Coupon{
		ID:    uuid.New(),
		Name:  req.Name,
		Code:  req.Code,
		Limit: req.Limit,
	}

	if err := gormDB.Create(&coupon).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create ticket.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Ticket created successfully.",
		"ticket_id": coupon.ID,
	})
}
