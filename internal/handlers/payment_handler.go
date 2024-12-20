package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/middleware"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xendit/xendit-go/v6/invoice"
	"github.com/xendit/xendit-go/v6/payout"
	"gorm.io/gorm"
)

type PaymentRequest struct {
	TicketID uuid.UUID  `json:"ticket_id" binding:"required"`
	CouponID *uuid.UUID `json:"coupon_id"`
	Quantity int        `json:"quantity" binding:"required,min=1"`
}

func CreatePaymentLink(c *gin.Context) {
	var paymentReq PaymentRequest
	if err := c.ShouldBindJSON(&paymentReq); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID type.")
		return
	}

	xenditClient := middleware.GetXenditClient(c)
	if xenditClient == nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Xendit client not initialized.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var ticket models.Ticket
	if err := gormDB.Preload("Event.Categories").First(&ticket, paymentReq.TicketID).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
		return
	}

	var user models.User
	if err := gormDB.First(&user, userUUID).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
		return
	}

	var categoryNames []string
	for _, category := range ticket.Event.Categories {
		categoryNames = append(categoryNames, category.Name)
	}
	categoriesStr := strings.Join(categoryNames, ",")

	totalAmount := int(ticket.Price * paymentReq.Quantity)
	var appliedCoupon int
	var usedCouponID *uuid.UUID

	if paymentReq.CouponID != nil {
		var coupon models.Coupon
		if err := gormDB.First(&coupon, paymentReq.CouponID).Error; err != nil {
			helpers.RespondWithError(c, http.StatusNotFound, "Coupon not found.")
			return
		}

		now := time.Now()
		if now.Before(coupon.ValidAt) || now.After(coupon.ExpiredAt) {
			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon is not currently valid.")
			return
		}

		var userCoupon models.UserCoupon
		err := gormDB.Where("user_id = ? AND coupon_id = ?", userUUID, *paymentReq.CouponID).First(&userCoupon).Error
		if err != nil {
			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon not claimed by user.")
			return
		}

		if userCoupon.IsUsed {
			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon has already been used.")
			return
		}

		appliedCoupon = int(coupon.Discount)
		totalAmount = totalAmount * (100 - appliedCoupon) / 100
		usedCouponID = &coupon.ID
	}

	adminFeePercent := 1.5
	adminFee := int(float64(totalAmount) * float64(adminFeePercent) / 100)

	descStr := fmt.Sprintf("%s - %s (Qty: %d)",
		ticket.Event.Title,
		ticket.Type,
		paymentReq.Quantity,
	)

	invoiceRequest := invoice.CreateInvoiceRequest{
		ExternalId:  fmt.Sprintf("INV-%d-%s", time.Now().Unix(), helpers.EncryptExternalID(ticket.ID, usedCouponID)),
		Amount:      float64(totalAmount + adminFee),
		PayerEmail:  &user.Email,
		Description: &descStr,
		Customer: &invoice.CustomerObject{
			GivenNames:   *invoice.NewNullableString(&user.Name),
			Email:        *invoice.NewNullableString(&user.Email),
			MobileNumber: *invoice.NewNullableString(&user.PhoneNumber),
		},
		Fees: []invoice.InvoiceFee{
			{
				Type:  fmt.Sprintf("Admin Fee (%.1f%%)", adminFeePercent),
				Value: float32(adminFee),
			},
		},
		Items: []invoice.InvoiceItem{
			{
				Name:     fmt.Sprintf("%s - %s", ticket.Event.Title, ticket.Type),
				Quantity: float32(paymentReq.Quantity),
				Price:    float32(totalAmount / paymentReq.Quantity),
				Category: &categoriesStr,
			},
		},
	}

	resp, _, err := xenditClient.InvoiceApi.CreateInvoice(context.Background()).CreateInvoiceRequest(invoiceRequest).Execute()
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create payment link.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_url": resp.InvoiceUrl,
	})
}

func PaymentNotification(c *gin.Context) {
	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	xenditClient := middleware.GetXenditClient(c)
	if xenditClient == nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Xendit client not initialized.")
		return
	}

	var payload *invoice.InvoiceCallback
	if err := c.ShouldBindJSON(&payload); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid notification payload.")
		return
	}

	if payload.Status != "PAID" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payment is not paid",
		})
	}

	if payload.Status == "PAID" {
		var user models.User
		if err := gormDB.Where("email = ?", payload.PayerEmail).First(&user).Error; err != nil {
			helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
			return
		}

		ticketID, couponID, _ := helpers.ExtractTicketID(payload.ExternalId)
		if ticketID == uuid.Nil {
			helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to extract IDs from external ID.")
			return
		}

		payment := models.Payment{
			Amount:        int(payload.Amount),
			Method:        *payload.PaymentMethod,
			Status:        payload.Status,
			UserID:        user.ID,
			CouponID:      couponID,
			TransactionID: payload.ExternalId,
		}

		if err := gormDB.Create(&payment).Error; err != nil {
			helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create payment.")
			return
		}

		for _, item := range payload.Items {
			for i := 0; i < int(item.Quantity); i++ {
				purchase := models.Purchase{
					TicketID:  ticketID,
					UserID:    payment.UserID,
					PaymentID: payment.ID,
					IsUsed:    false,
				}

				if err := gormDB.Create(&purchase).Error; err != nil {
					helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create purchase.")
					return
				}
			}
		}

		if couponID != nil {
			if err := gormDB.Model(&models.UserCoupon{}).Where("user_id = ? AND coupon_id = ?", user.ID, *couponID).Update("is_used", true).Error; err != nil {
				helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update user coupon.")
				return
			}
		}

		var Ticket models.Ticket
		if err := gormDB.Preload("Event.User").Where("id = ?", ticketID).First(&Ticket).Error; err != nil {
			helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
			return
		}

		totalFee := float64(0)
		for _, fee := range payload.Fees {
			totalFee += float64(fee.Value)
		}

		idempotencyKey := fmt.Sprintf("disb-%s", uuid.New().String())
		payoutRequest := *payout.NewCreatePayoutRequest(
			fmt.Sprintf("disb-%s", payment.ID),
			*Ticket.Event.User.AccountChannel,
			payout.DigitalPayoutChannelProperties{
				AccountHolderName: *payout.NewNullableString(Ticket.Event.User.AccountName),
				AccountNumber:     *Ticket.Event.User.AccountNumber,
			},
			float32(payload.Amount-totalFee),
			"IDR",
		)

		resp, _, err := xenditClient.PayoutApi.CreatePayout(context.Background()).
			IdempotencyKey(idempotencyKey).
			CreatePayoutRequest(payoutRequest).
			Execute()

		if err != nil {
			helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create payout.")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Payment created to channel: %s", resp.Payout.ChannelCode),
		})
	}
}
