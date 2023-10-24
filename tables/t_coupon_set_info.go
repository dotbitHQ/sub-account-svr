package tables

import (
	"time"
)

type CouponSetInfo struct {
	Id            int64     `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid           string    `gorm:"column:cid;default:;NOT NULL"`
	AccountId     string    `gorm:"column:account_id;default:;NOT NULL"`
	ManagerAid    int16     `gorm:"column:manager_aid;default:0;NOT NULL"`
	ManagerSubAid int16     `gorm:"column:manager_sub_aid;default:0;NOT NULL"`
	Manager       string    `gorm:"column:manager;default:;NOT NULL"`
	Root          string    `gorm:"column:root;default:;NOT NULL"`
	Name          string    `gorm:"column:name;default:;NOT NULL"`
	Note          string    `gorm:"column:note;default:;NOT NULL"`
	Price         string    `gorm:"column:price;default:;NOT NULL"`
	Num           int       `gorm:"column:num;default:0;NOT NULL"`
	ExpiredAt     int64     `gorm:"column:expired_at;default:0;NOT NULL"`
	Status        int       `gorm:"column:status;default:0;NOT NULL"`
	Signature     string    `gorm:"column:signature;default:;NOT NULL"`
	CreatedAt     time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt     time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (t *CouponSetInfo) TableName() string {
	return "t_coupon_set_info"
}
