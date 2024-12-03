package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Coupon struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Name        string    `gorm:"not null"`
	Code        *string   `gorm:"unique"`
	Limit       int       `gorm:"not null"`
	Discount    int       `gorm:"not null"`
	Description *string
	ValidAt     time.Time `gorm:"not null"`
	ExpiredAt   time.Time `gorm:"not null"`
	Users       []User    `gorm:"many2many:user_coupons;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type UserCoupon struct {
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	CouponID  uuid.UUID `gorm:"type:uuid;not null;index"`
	IsUsed    bool      `gorm:"not null;default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserCoupon) TableName() string {
	return "user_coupons"
}
