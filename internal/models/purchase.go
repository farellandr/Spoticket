package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Purchase struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	Total     int       `gorm:"not null"`
	IsUsed    bool      `gorm:"not null;default:false"`
	TicketId  uuid.UUID
	Ticket    Ticket
	UserId    uuid.UUID
	User      User
	PaymentId uuid.UUID
	Payment   Payment
}

func (purchase *Purchase) BeforeCreate(tx *gorm.DB) (err error) {
	if purchase.ID == uuid.Nil {
		purchase.ID = uuid.New()
	}
	return
}
