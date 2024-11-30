package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Coupon struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Code      string    `gorm:"not null;unique"`
	Discount  int       `gorm:"not null"`
	Users     []User    `gorm:"many2many:user_coupons;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
