package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/middleware"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xendit/xendit-go/v6/invoice"
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

	adminFeePercent := 1.5
	adminFee := int(float64(totalAmount) * float64(adminFeePercent) / 100)

	descStr := fmt.Sprintf("%s - %s (Qty: %d)",
		ticket.Event.Title,
		ticket.Type,
		paymentReq.Quantity,
	)

	invoiceRequest := invoice.CreateInvoiceRequest{
		ExternalId:  fmt.Sprintf("INV-%d-%s", time.Now().Unix(), ticket.ID),
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

	resp, r, err := xenditClient.InvoiceApi.CreateInvoice(context.Background()).CreateInvoiceRequest(invoiceRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `InvoiceApi.CreateInvoice``: %v\n", err.Error())

		b, _ := json.Marshal(err.FullError())
		fmt.Fprintf(os.Stderr, "Full Error Struct: %v\n", string(b))

		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_url": resp.InvoiceUrl,
	})
}

func PaymentNotification(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid notification payload",
		})
		return
	}

	status, ok := payload["status"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract transaction status",
		})
		return
	}

	externalID, ok := payload["external_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract external ID",
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

	additionalData, ok := payload["additional_data"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract additional data",
		})
		return
	}

	userIDStr, ok := additionalData["userId"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract user ID",
		})
		return
	}

	ticketIDStr, ok := additionalData["ticketId"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to extract ticket ID",
		})
		return
	}

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
	if couponIDStr, ok := additionalData["couponId"].(string); ok && couponIDStr != "" {
		parsedCouponID, err := uuid.Parse(couponIDStr)
		if err == nil {
			couponID = &parsedCouponID
		}
	}

	if status == "PAID" {
		amount, ok := payload["amount"].(float64)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to extract payment amount",
			})
			return
		}

		paymentMethod, ok := payload["payment_method"].(string)
		if !ok {
			paymentMethod = "XENDIT"
		}

		payment := models.Payment{
			Amount:        int(amount),
			Method:        paymentMethod,
			Status:        status,
			TransactionID: externalID,
			UserID:        userUUID,
			CouponID:      couponID,
		}

		err = gormDB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&payment).Error; err != nil {
				return err
			}

			lineItems, ok := payload["items"].([]interface{})
			if !ok {
				return fmt.Errorf("unable to extract line items")
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

// func CreatePaymentLink(c *gin.Context) {
// 	var paymentReq PaymentRequest
// 	if err := c.ShouldBindJSON(&paymentReq); err != nil {
// 		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
// 		return
// 	}

// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		helpers.RespondWithError(c, http.StatusUnauthorized, "User ID not found in token.")
// 		return
// 	}
// 	userUUID, ok := userID.(uuid.UUID)
// 	if !ok {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID type.")
// 		return
// 	}

// 	db, exists := c.Get("db")
// 	if !exists {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
// 		return
// 	}
// 	gormDB := db.(*gorm.DB)

// 	var ticket models.Ticket
// 	if err := gormDB.Preload("Event.Categories").First(&ticket, paymentReq.TicketID).Error; err != nil {
// 		helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
// 		return
// 	}

// 	var user models.User
// 	if err := gormDB.First(&user, userUUID).Error; err != nil {
// 		helpers.RespondWithError(c, http.StatusNotFound, "User not found.")
// 		return
// 	}

// 	var categoryNames []string
// 	for _, category := range ticket.Event.Categories {
// 		categoryNames = append(categoryNames, category.Name)
// 	}
// 	categoriesStr := strings.Join(categoryNames, ",")

// 	totalAmount := int(ticket.Price * paymentReq.Quantity)
// 	var appliedCoupon int

// 	if paymentReq.CouponID != nil {
// 		var coupon models.Coupon
// 		if err := gormDB.First(&coupon, paymentReq.CouponID).Error; err != nil {
// 			helpers.RespondWithError(c, http.StatusNotFound, "Coupon not found.")
// 			return
// 		}

// 		now := time.Now()
// 		if now.Before(coupon.ValidAt) || now.After(coupon.ExpiredAt) {
// 			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon is not currently valid.")
// 			return
// 		}

// 		var userCoupon models.UserCoupon
// 		err := gormDB.Where("user_id = ? AND coupon_id = ?", userUUID, *paymentReq.CouponID).First(&userCoupon).Error
// 		if err != nil {
// 			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon not claimed by user.")
// 			return
// 		}

// 		if userCoupon.IsUsed {
// 			helpers.RespondWithError(c, http.StatusBadRequest, "Coupon has already been used.")
// 			return
// 		}

// 		appliedCoupon = int(coupon.Discount)
// 		totalAmount = totalAmount * (100 - appliedCoupon) / 100
// 	}

// 	paymentBody := map[string]interface{}{
// 		"order": map[string]interface{}{
// 			"amount":                totalAmount,
// 			"invoice_number":        fmt.Sprintf("INV-%d", time.Now().Unix()),
// 			"currency":              "IDR",
// 			"language":              "EN",
// 			"auto_redirect":         false,
// 			"disable_retry_payment": true,
// 			"line_items": []map[string]interface{}{
// 				{
// 					"id":       "001",
// 					"name":     fmt.Sprintf("%s - %s", ticket.Event.Title, ticket.Type),
// 					"quantity": paymentReq.Quantity,
// 					"price":    int(totalAmount / paymentReq.Quantity),
// 					"category": categoriesStr,
// 					"type":     ticket.Type,
// 				},
// 			},
// 		},
// 		"payment": map[string]interface{}{
// 			"payment_due_date": 10,
// 		},
// 		"customer": map[string]interface{}{
// 			"id":    userUUID.String(),
// 			"name":  user.Name,
// 			"phone": user.PhoneNumber,
// 			"email": user.Email,
// 		},
// 		"additional_info": map[string]interface{}{
// 			"override_notification_url": fmt.Sprintf("https://f209kkb4-3222.asse.devtunnels.ms/v1/payments/notification?userId=%s&ticketId=%s&couponId=%s",
// 				userUUID.String(),
// 				ticket.ID.String(),
// 				paymentReq.CouponID,
// 			),
// 		},
// 	}

// 	jsonBody, err := json.Marshal(paymentBody)
// 	if err != nil {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare payment request.")
// 		return
// 	}

// 	headerGenerator := helpers.NewDokuHeaderGenerator(
// 		os.Getenv("DOKU_CLIENT_ID"),
// 		os.Getenv("DOKU_SECRET_KEY"),
// 		"/checkout/v1/payment",
// 	)
// 	headers := headerGenerator.GetHeaders(string(jsonBody))

// 	httpReq, err := http.NewRequest("POST", "https://api-sandbox.doku.com/checkout/v1/payment", bytes.NewBuffer(jsonBody))
// 	if err != nil {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create payment request.")
// 		return
// 	}

// 	for key, value := range headers {
// 		httpReq.Header.Set(key, value)
// 	}

// 	client := &http.Client{}
// 	resp, err := client.Do(httpReq)
// 	if err != nil {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to send payment request.")
// 		return
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to read payment response.")
// 		return
// 	}

// 	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
// 		helpers.RespondWithError(c, resp.StatusCode, "Payment link generation failed.")
// 		return
// 	}

// 	var responseBody map[string]interface{}
// 	if err := json.Unmarshal(body, &responseBody); err != nil {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to parse payment response.")
// 		return
// 	}

// 	paymentURL, ok := responseBody["response"].(map[string]interface{})["payment"].(map[string]interface{})["url"].(string)
// 	if !ok {
// 		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to extract payment URL.")
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"payment_url": paymentURL,
// 	})
// }

// func PaymentNotification(c *gin.Context) {
// 	userIDStr := c.Query("userId")
// 	ticketIDStr := c.Query("ticketId")
// 	couponIDStr := c.Query("couponId")

// 	userUUID, err := uuid.Parse(userIDStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "Invalid user ID",
// 		})
// 		return
// 	}

// 	ticketID, err := uuid.Parse(ticketIDStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "Invalid ticket ID",
// 		})
// 		return
// 	}

// 	var couponID *uuid.UUID
// 	if couponIDStr != "" {
// 		parsedCouponID, err := uuid.Parse(couponIDStr)
// 		if err == nil {
// 			couponID = &parsedCouponID
// 		}
// 	}

// 	var notificationPayload map[string]interface{}
// 	if err := c.ShouldBindJSON(&notificationPayload); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "Invalid notification payload",
// 		})
// 		return
// 	}

// 	transactionStatus, ok := notificationPayload["transaction"].(map[string]interface{})["status"].(string)
// 	if !ok {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "Unable to extract transaction status",
// 		})
// 		return
// 	}

// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error": "Database connection not found",
// 		})
// 		return
// 	}
// 	gormDB := db.(*gorm.DB)

// 	var user models.User
// 	if err := gormDB.First(&user, userUUID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"error": "User not found",
// 		})
// 		return
// 	}

// 	var ticket models.Ticket
// 	if err := gormDB.First(&ticket, ticketID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"error": "Ticket not found",
// 		})
// 		return
// 	}

// 	if transactionStatus == "SUCCESS" {
// 		orderInfo, ok := notificationPayload["order"].(map[string]interface{})
// 		if !ok {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": "Unable to extract order information",
// 			})
// 			return
// 		}

// 		amount, ok := orderInfo["amount"].(float64)
// 		if !ok {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": "Unable to extract payment amount",
// 			})
// 			return
// 		}

// 		transactionID, ok := notificationPayload["order"].(map[string]interface{})["invoice_number"].(string)
// 		if !ok {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": "Unable to extract transaction ID",
// 			})
// 			return
// 		}

// 		paymentMethod, ok := notificationPayload["channel"].(map[string]interface{})["id"].(string)
// 		if !ok {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": "Unable to extract payment method",
// 			})
// 			return
// 		}

// 		lineItems, ok := notificationPayload["additional_info"].(map[string]interface{})["line_items"].([]interface{})
// 		if !ok {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": "Unable to extract line items",
// 			})
// 			return
// 		}

// 		payment := models.Payment{
// 			Amount:        int(amount),
// 			Method:        paymentMethod,
// 			Status:        transactionStatus,
// 			TransactionID: transactionID,
// 			UserID:        userUUID,
// 			CouponID:      couponID,
// 		}

// 		err = gormDB.Transaction(func(tx *gorm.DB) error {
// 			if err := tx.Create(&payment).Error; err != nil {
// 				return err
// 			}

// 			for _, item := range lineItems {
// 				lineItem, ok := item.(map[string]interface{})
// 				if !ok {
// 					return fmt.Errorf("invalid line item format")
// 				}

// 				quantityFloat, ok := lineItem["quantity"].(float64)
// 				if !ok {
// 					return fmt.Errorf("unable to extract quantity")
// 				}
// 				quantity := int(quantityFloat)

// 				for i := 0; i < quantity; i++ {
// 					purchase := models.Purchase{
// 						Total:     int(amount) / quantity,
// 						TicketID:  ticketID,
// 						UserID:    userUUID,
// 						PaymentID: payment.ID,
// 						IsUsed:    false,
// 					}

// 					if err := tx.Create(&purchase).Error; err != nil {
// 						return err
// 					}
// 				}
// 			}

// 			if couponID != nil {
// 				return tx.Model(&models.UserCoupon{}).
// 					Where("user_id = ? AND coupon_id = ?", userUUID, *couponID).
// 					Update("is_used", true).Error
// 			}

// 			return nil
// 		})

// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"error": "Failed to create payment and purchases",
// 			})
// 			return
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Notification processed successfully",
// 	})
// }
