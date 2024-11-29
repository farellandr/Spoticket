package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	Email       string    `gorm:"unique;not null"`
	Password    string    `gorm:"not null"`
	PhoneNumber string    `gorm:"not null"`
	RoleID      uuid.UUID
	Role        Role
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return
}
