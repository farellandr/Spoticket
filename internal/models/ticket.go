package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ticket struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Type      string    `gorm:"not null"`
	Price     int       `gorm:"not null"`
	Limit     *int
	EventID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Event     *Event     `gorm:"foreignKey:EventID"`
	Purchases []Purchase `gorm:"foreignKey:TicketID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
