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
	Discount    int       `json:"discount" binding:"required"`
	ValidAt     time.Time `json:"valid_at" binding:"required"`
	ExpiredAt   time.Time `json:"expired_at" binding:"required"`
	Description *string   `json:"description"`
}

type ClaimCouponRequest struct {
	Code string `json:"code" binding:"required"`
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

	coupon := models.Coupon{
		ID:          uuid.New(),
		Name:        req.Name,
		Code:        req.Code,
		Limit:       req.Limit,
		Discount:    req.Discount,
		ValidAt:     req.ValidAt,
		ExpiredAt:   req.ExpiredAt,
		Description: req.Description,
	}

	if err := gormDB.Create(&coupon).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create coupon.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Coupon created successfully.",
		"coupon_id": coupon.ID,
	})
}

func ClaimCoupon(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User not authenticated.")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format.")
		return
	}

	var request ClaimCouponRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Coupon code is required.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var coupon models.Coupon
	if err := gormDB.Where("code = ?", request.Code).First(&coupon).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "Coupon not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving coupon.")
		return
	}

	if time.Now().After(coupon.ExpiredAt) {
		helpers.RespondWithError(c, http.StatusBadRequest, "Coupon has expired.")
		return
	}

	var usageCount int64
	gormDB.Model(&models.UserCoupon{}).Where("coupon_id = ?", coupon.ID).Count(&usageCount)
	if int(usageCount) >= coupon.Limit {
		helpers.RespondWithError(c, http.StatusBadRequest, "Coupon usage limit reached.")
		return
	}

	var existingClaim models.UserCoupon
	if err := gormDB.Where("user_id = ? AND coupon_id = ?", userUUID, coupon.ID).First(&existingClaim).Error; err == nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "You have already claimed this coupon.")
		return
	}

	userCoupon := models.UserCoupon{
		UserID:   userUUID,
		CouponID: coupon.ID,
		IsUsed:   false,
	}
	if err := gormDB.Create(&userCoupon).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to claim the coupon.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupon claimed successfully.",
		"coupon": gin.H{
			"id":         coupon.ID,
			"name":       coupon.Name,
			"discount":   coupon.Discount,
			"valid_at":   coupon.ValidAt,
			"expired_at": coupon.ExpiredAt,
		},
	})
}

func ListCoupons(c *gin.Context) {
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

	query := gormDB.Model(&models.Coupon{})
	var totalCount int64
	query.Count(&totalCount)

	var coupons []models.Coupon
	offset := (pageNum - 1) * limitNum
	err = query.Offset(offset).Limit(limitNum).Order("created_at DESC").Find(&coupons).Error
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving coupons.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coupons":     coupons,
		"total":       totalCount,
		"page":        pageNum,
		"limit":       limitNum,
		"total_pages": (totalCount + int64(limitNum) - 1) / int64(limitNum),
	})
}

func GetCoupon(c *gin.Context) {
	couponID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var coupon models.Coupon
	if err := gormDB.Where("id = ?", couponID).First(&coupon).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "Coupon not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving coupon.")
		return
	}

	c.JSON(http.StatusOK, coupon)
}

func UpdateCoupon(c *gin.Context) {
	couponID := c.Param("id")

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

	var coupon models.Coupon
	if err := gormDB.Where("id = ?", couponID).First(&coupon).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusForbidden, "Coupon not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error finding coupon.")
		return
	}

	coupon.Name = req.Name
	coupon.Code = req.Code
	coupon.Limit = req.Limit
	coupon.Discount = req.Discount
	coupon.ValidAt = req.ValidAt
	coupon.ExpiredAt = req.ExpiredAt
	coupon.Description = req.Description

	if err := gormDB.Save(&coupon).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update coupon.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupon updated successfully.",
		"coupon":  coupon,
	})
}

func DeleteCoupon(c *gin.Context) {
	couponID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	result := gormDB.Where("id = ?", couponID).Delete(&models.Coupon{})
	if result.Error != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to delete coupon.")
		return
	}

	if result.RowsAffected == 0 {
		helpers.RespondWithError(c, http.StatusForbidden, "Coupon not found or you don't have permission to delete.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupon deleted successfully.",
	})
}
