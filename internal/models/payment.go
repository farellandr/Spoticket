package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	ID            uuid.UUID `gorm:"type:uuid;primary_key"`
	Amount        int       `gorm:"not null"`
	Method        string    `gorm:"not null"`
	Status        string    `gorm:"not null;default:'pending'"`
	TransactionId string    `gorm:"not null"`
	UserId        uuid.UUID
	User          User
}

func (payment *Payment) BeforeCreate(tx *gorm.DB) (err error) {
	if payment.ID == uuid.Nil {
		payment.ID = uuid.New()
	}
	return
}
