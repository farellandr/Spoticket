package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Purchase struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Total     int       `gorm:"not null"`
	IsUsed    bool      `gorm:"not null;default:false"`
	TicketID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Ticket    Ticket    `gorm:"foreignKey:TicketID"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	User      User      `gorm:"foreignKey:UserID"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`
	Payment   Payment   `gorm:"foreignKey:PaymentID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
