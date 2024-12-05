package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Amount        int        `gorm:"not null"`
	Method        string     `gorm:"not null"`
	Status        string     `gorm:"not null;default:'pending'"`
	TransactionID string     `gorm:"not null"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	User          *User      `gorm:"foreignKey:UserID"`
	CouponID      *uuid.UUID `gorm:"type:uuid"`
	Coupon        *Coupon    `gorm:"foreignKey:CouponID"`
	Purchase      *Purchase  `gorm:"foreignKey:PaymentID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
