package tables

import (
	"time"
)

type CouponStatus int

const (
	CouponStatusNormal CouponStatus = 0
	CouponStatusUsed   CouponStatus = 1
)

type CouponCode string

type CouponInfo struct {
	Id        uint64       `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid       string       `gorm:"index:idx_cid;column:cid;default:;NOT NULL"`
	Code      string       `gorm:"index:idx_code;column:code;default:;NOT NULL"`
	Status    CouponStatus `gorm:"column:status;default:0;NOT NULL"`
	CreatedAt time.Time    `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt time.Time    `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (t *CouponInfo) TableName() string {
	return "t_coupon_info"
}
