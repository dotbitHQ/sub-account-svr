package tables

import (
	"das_sub_account/config"
	"das_sub_account/encrypt"
	"database/sql/driver"
	"fmt"
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
	Cid       string       `gorm:"column:cid;default:;NOT NULL"`
	Code      *CouponCode  `gorm:"column:code;default:;NOT NULL"`
	Status    CouponStatus `gorm:"column:status;default:0;NOT NULL"`
	CreatedAt time.Time    `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt time.Time    `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (t *CouponInfo) TableName() string {
	return "t_coupon_info"
}

func (j *CouponCode) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %s", value)
	}
	code, err := encrypt.AesDecrypt(string(bytes), config.Cfg.Das.Coupon.EncryptionKey)
	if err != nil {
		return err
	}
	*j = CouponCode(code)
	return nil
}

func (j *CouponCode) Value() (driver.Value, error) {
	if len(*j) == 0 {
		return nil, nil
	}
	code, err := encrypt.AesEncrypt(string(*j), config.Cfg.Das.Coupon.EncryptionKey)
	if err != nil {
		return nil, err
	}
	return []byte(code), nil
}
