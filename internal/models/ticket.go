package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ticket struct {
	gorm.Model
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	Type    string    `gorm:"not null"`
	Price   int       `gorm:"not null"`
	Limit   *int
	EventID uuid.UUID
	Event   Event
}

func (ticket *Ticket) BeforeCreate(tx *gorm.DB) (err error) {
	if ticket.ID == uuid.Nil {
		ticket.ID = uuid.New()
	}
	return
}
