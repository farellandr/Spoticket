package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func generateQRCodeData(purchase *models.Purchase) string {
	return fmt.Sprintf("purchase:%s:ticket:%s:event:%s",
		purchase.ID.String(),
		purchase.TicketID.String(),
		purchase.Ticket.EventID.String(),
	)
}

func GenerateTicketQR(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User not authenticated.")
		return
	}

	purchaseIDStr := c.Param("purchaseId")
	purchaseID, err := uuid.Parse(purchaseIDStr)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid purchase ID")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found")
		return
	}
	gormDB := db.(*gorm.DB)

	var purchase models.Purchase
	if err := gormDB.Preload("Ticket.Event").First(&purchase, purchaseID).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "Purchase not found")
		return
	}

	if purchase.UserID != userID {
		helpers.RespondWithError(c, http.StatusForbidden, "You don't have permission to generate QR code for this purchase")
		return
	}

	qrData := generateQRCodeData(&purchase)

	qrImage, err := qrcode.Encode(qrData, qrcode.Medium, 1444)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to generate QR code")
		return
	}

	c.Data(http.StatusOK, "image/png", qrImage)
}

func extractPurchaseIDFromQRData(qrData string) (uuid.UUID, error) {
	parts := strings.Split(qrData, ":")
	if len(parts) != 6 || parts[0] != "purchase" {
		return uuid.Nil, fmt.Errorf("invalid QR data format")
	}
	return uuid.Parse(parts[1])
}

func ValidateTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User not authenticated.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found")
		return
	}
	gormDB := db.(*gorm.DB)

	var validationRequest struct {
		QRData string `json:"qr_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&validationRequest); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	purchaseID, err := extractPurchaseIDFromQRData(validationRequest.QRData)
	if err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid QR code format")
		return
	}

	var purchase models.Purchase
	if err := gormDB.Preload("Ticket.Event").First(&purchase, purchaseID).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "Purchase not found")
		return
	}

	if purchase.Ticket.Event.UserID != userID {
		helpers.RespondWithError(c, http.StatusForbidden, "You don't have permission to validate this ticket")
		return
	}

	if validationRequest.QRData != generateQRCodeData(&purchase) {
		helpers.RespondWithError(c, http.StatusForbidden, "Invalid QR code")
		return
	}

	if purchase.IsUsed {
		helpers.RespondWithError(c, http.StatusForbidden, "Ticket already used")
		return
	}

	if err := gormDB.Model(&purchase).Update("is_used", true).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to validate ticket")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket validated successfully",
		"ticket": gin.H{
			"event_title": purchase.Ticket.Event.Title,
			"ticket_type": purchase.Ticket.Type,
		},
	})
}
