package tables

import (
	"time"
)

const (
	CouponStatusNormal = 0
	CouponStatusUsed   = 1
)

type CouponInfo struct {
	Id        uint64    `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid       string    `gorm:"column:cid;default:;NOT NULL"`
	Code      string    `gorm:"column:code;default:;NOT NULL"`
	Status    int       `gorm:"column:status;default:0;NOT NULL"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (t *CouponInfo) TableName() string {
	return "t_coupon_info"
}
