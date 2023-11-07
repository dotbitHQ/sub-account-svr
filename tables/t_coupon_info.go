package tables

import (
	"time"
)

type CouponStatus int

const (
	CouponStatusNormal     CouponStatus = 0
	CouponStatusUsed       CouponStatus = 1
	CouponStatusDeactivate CouponStatus = 2
)

type CouponCode string

type CouponInfo struct {
	Id        uint64       `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid       string       `gorm:"index:idx_cid;column:cid;default:;NOT NULL"`
	Code      string       `gorm:"uniqueIndex:idx_code;column:code;default:;NOT NULL"`
	Status    CouponStatus `gorm:"column:status;default:0;NOT NULL"`
	CreatedAt time.Time    `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

func (t *CouponInfo) TableName() string {
	return "t_coupon_info"
}
