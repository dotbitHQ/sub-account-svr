package tables

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type CouponSetInfo struct {
	Id            int64           `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid           string          `gorm:"column:cid;default:;NOT NULL"`
	OrderId       string          `gorm:"column:order_id;default:;NOT NULL"`
	AccountId     string          `gorm:"column:account_id;default:;NOT NULL"`
	Account       string          `gorm:"column:account;default:;NOT NULL"`
	ManagerAid    int             `gorm:"column:manager_aid;default:0;NOT NULL"`
	ManagerSubAid int             `gorm:"column:manager_sub_aid;default:0;NOT NULL"`
	Manager       string          `gorm:"column:manager;default:;NOT NULL"`
	Root          string          `gorm:"column:root;default:;NOT NULL"`
	Name          string          `gorm:"column:name;default:;NOT NULL"`
	Note          string          `gorm:"column:note;default:;NOT NULL"`
	Price         decimal.Decimal `gorm:"price:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	Num           int             `gorm:"column:num;default:0;NOT NULL"`
	ExpiredAt     int64           `gorm:"column:expired_at;default:0;NOT NULL"`
	Status        int             `gorm:"column:status;default:0;NOT NULL"`
	Signature     string          `gorm:"column:signature;default:;NOT NULL"`
	CreatedAt     time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}

func (t *CouponSetInfo) TableName() string {
	return "t_coupon_set_info"
}

func (t *CouponSetInfo) InitCid() {
	uid, _ := uuid.NewUUID()
	t.Cid = strings.ReplaceAll(uid.String(), "-", "")
}
