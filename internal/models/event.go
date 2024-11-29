package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	StartTime   time.Time `gorm:"not null"`
	EndTime     time.Time `gorm:"not null"`
	Province    string    `gorm:"not null"`
	City        string    `gorm:"not null"`
	District    string    `gorm:"not null"`
	SubDistrict string    `gorm:"not null"`
	Location    string    `gorm:"not null"`
	User        User
	UserID      uuid.UUID
}

func (event *Event) BeforeCreate(tx *gorm.DB) (err error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return
}
