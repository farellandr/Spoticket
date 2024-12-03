package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Title       string     `gorm:"not null"`
	Description string     `gorm:"not null"`
	StartTime   time.Time  `gorm:"not null"`
	EndTime     time.Time  `gorm:"not null"`
	Province    string     `gorm:"not null"`
	City        string     `gorm:"not null"`
	District    string     `gorm:"not null"`
	SubDistrict string     `gorm:"not null"`
	Location    string     `gorm:"not null"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	User        *User      `gorm:"foreignKey:UserID"`
	Categories  []Category `gorm:"many2many:event_categories;"`
	Tickets     []Ticket   `gorm:"foreignKey:EventID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
