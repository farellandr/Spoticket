package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Name           string     `gorm:"not null"`
	Email          string     `gorm:"unique;not null"`
	Password       string     `gorm:"not null"`
	PhoneNumber    string     `gorm:"not null"`
	RoleID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role           *Role      `gorm:"foreignKey:RoleID"`
	Events         []Event    `gorm:"foreignKey:UserID"`
	Purchases      []Purchase `gorm:"foreignKey:UserID"`
	Payments       []Payment  `gorm:"foreignKey:UserID"`
	Coupons        []Coupon   `gorm:"many2many:user_coupons;"`
	ProfilePicture *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
