package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	gorm.Model
	ID   uuid.UUID `gorm:"type:uuid;primary_key"`
	Name string    `gorm:"unique;not null"`
}

func (role *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	return
}
