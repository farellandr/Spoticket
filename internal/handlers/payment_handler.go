package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	}

	paymentBody := map[string]interface{}{
		"order": map[string]interface{}{
			"amount":                totalAmount,
			"invoice_number":        fmt.Sprintf("INV-%d", time.Now().Unix()),
			"currency":              "IDR",
			"language":              "EN",
			"auto_redirect":         false,
			"disable_retry_payment": true,
			"line_items": []map[string]interface{}{
				{
					"id":       "001",
					"name":     fmt.Sprintf("%s - %s", ticket.Event.Title, ticket.Type),
					"quantity": paymentReq.Quantity,
					"price":    int(totalAmount / paymentReq.Quantity),
					"category": categoriesStr,
					"type":     ticket.Type,
				},
			},
		},
		"payment": map[string]interface{}{
			"payment_due_date": 10,
		},
		"customer": map[string]interface{}{
			"id":    userUUID.String(),
			"name":  user.Name,
			"phone": user.PhoneNumber,
			"email": user.Email,
		},
		"additional_info": map[string]interface{}{
			"override_notification_url": fmt.Sprintf("https://f209kkb4-3222.asse.devtunnels.ms/v1/payments/notification?userId=%s&ticketId=%s&couponId=%s",
				userUUID.String(),
				ticket.ID.String(),
				paymentReq.CouponID,
			),
		},
	}

	jsonBody, err := json.Marshal(paymentBody)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare payment request.")
		return
	}

	headerGenerator := helpers.NewDokuHeaderGenerator(
		os.Getenv("DOKU_CLIENT_ID"),
		os.Getenv("DOKU_SECRET_KEY"),
		"/checkout/v1/payment",
	)
	headers := headerGenerator.GetHeaders(string(jsonBody))

	httpReq, err := http.NewRequest("POST", "https://api-sandbox.doku.com/checkout/v1/payment", bytes.NewBuffer(jsonBody))
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create payment request.")
		return
	}

	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to send payment request.")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to read payment response.")
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		helpers.RespondWithError(c, resp.StatusCode, "Payment link generation failed.")
		return
	}

	var responseBody map[string]interface{}
	if err := json.Unmarshal(body, &responseBody); err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to parse payment response.")
		return
	}

	paymentURL, ok := responseBody["response"].(map[string]interface{})["payment"].(map[string]interface{})["url"].(string)
	if !ok {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to extract payment URL.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_url": paymentURL,
	})
}

func PaymentNotification(c *gin.Context) {
	userIDStr := c.Query("userId")
	ticketIDStr := c.Query("ticketId")
	couponIDStr := c.Query("couponId")

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ticket ID",
		})
		return
	}

	var couponID *uuid.UUID
	if couponIDStr != "" {
		parsedCouponID, err := uuid.Parse(couponIDStr)
		if err == nil {
			couponID = &parsedCouponID
		}
	}

	var notificationPayload map[string]interface{}
	if err := c.ShouldBindJSON(&notificationPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid notification payload",
		})
		return
	}

	transactionStatus, ok := notificationPayload["transaction"].(map[string]interface{})["status"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract transaction status",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database connection not found",
		})
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.First(&user, userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	var ticket models.Ticket
	if err := gormDB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ticket not found",
		})
		return
	}

	if transactionStatus == "SUCCESS" {
		orderInfo, ok := notificationPayload["order"].(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract order information",
			})
			return
		}

		amount, ok := orderInfo["amount"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract payment amount",
			})
			return
		}

		transactionID, ok := notificationPayload["order"].(map[string]interface{})["invoice_number"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract transaction ID",
			})
			return
		}

		paymentMethod, ok := notificationPayload["channel"].(map[string]interface{})["id"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract payment method",
			})
			return
		}

		lineItems, ok := notificationPayload["additional_info"].(map[string]interface{})["line_items"].([]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract line items",
			})
			return
		}

		payment := models.Payment{
			Amount:        int(amount),
			Method:        paymentMethod,
			Status:        transactionStatus,
			TransactionID: transactionID,
			UserID:        userUUID,
			CouponID:      couponID,
		}

		err = gormDB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&payment).Error; err != nil {
				return err
			}

			for _, item := range lineItems {
				lineItem, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid line item format")
				}

				quantityFloat, ok := lineItem["quantity"].(float64)
				if !ok {
					return fmt.Errorf("unable to extract quantity")
				}
				quantity := int(quantityFloat)

				for i := 0; i < quantity; i++ {
					purchase := models.Purchase{
						Total:     int(amount) / quantity,
						TicketID:  ticketID,
						UserID:    userUUID,
						PaymentID: payment.ID,
						IsUsed:    false,
					}

					if err := tx.Create(&purchase).Error; err != nil {
						return err
					}
				}
			}

			if couponID != nil {
				return tx.Model(&models.UserCoupon{}).
					Where("user_id = ? AND coupon_id = ?", userUUID, *couponID).
					Update("is_used", true).Error
			}

			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create payment and purchases",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification processed successfully",
	})
}
