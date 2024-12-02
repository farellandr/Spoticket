package handlers

import (
	"net/http"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketRequest struct {
	Type    string    `json:"type" binding:"required"`
	Price   int       `json:"price" binding:"required"`
	Limit   *int      `json:"limit"`
	EventID uuid.UUID `json:"event_id" binding:"required"`
}

func CreateTicket(c *gin.Context) {
	var req TicketRequest
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

	var event models.Event
	if err := gormDB.Where("id = ? AND user_id = ?", req.EventID, userID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusForbidden, "Event not found or you don't have permission to modify it.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error verifying event ownership.")
		return
	}

	ticket := models.Ticket{
		ID:      uuid.New(),
		Type:    req.Type,
		Price:   req.Price,
		Limit:   req.Limit,
		EventID: req.EventID,
	}

	if err := gormDB.Create(&ticket).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create ticket.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Ticket created successfully.",
		"ticket_id": ticket.ID,
	})
}

func GetTicket(c *gin.Context) {
	ticketID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var ticket models.Ticket
	if err := gormDB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
			return
		}
		helpers.RespondWithError(c, http.StatusInternalServerError, "Error retrieving ticket.")
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func UpdateTicket(c *gin.Context) {
	ticketID := c.Param("id")
	var req TicketRequest
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

	var ticket models.Ticket
	if err := gormDB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
		return
	}

	var event models.Event
	if err := gormDB.Where("id = ? AND user_id = ?", ticket.EventID, userID).First(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusForbidden, "You don't have permission to modify this ticket.")
		return
	}

	ticket.Type = req.Type
	ticket.Price = req.Price
	ticket.Limit = req.Limit

	if err := gormDB.Save(&ticket).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to update ticket.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket updated successfully.",
		"ticket":  ticket,
	})
}

func DeleteTicket(c *gin.Context) {
	ticketID := c.Param("id")

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

	var ticket models.Ticket
	if err := gormDB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		helpers.RespondWithError(c, http.StatusNotFound, "Ticket not found.")
		return
	}

	var event models.Event
	if err := gormDB.Where("id = ? AND user_id = ?", ticket.EventID, userID).First(&event).Error; err != nil {
		helpers.RespondWithError(c, http.StatusForbidden, "You don't have permission to delete this ticket.")
		return
	}

	if err := gormDB.Delete(&ticket).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to delete ticket.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket deleted successfully.",
	})
}
