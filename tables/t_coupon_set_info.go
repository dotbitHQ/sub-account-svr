package tables

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

const (
	CouponSetInfoStatusCreated = 0
	CouponSetInfoStatusPaid    = 1
	CouponSetInfoStatusFailed  = 2
	CouponSetInfoStatusSuccess = 3
)

type CouponSetInfo struct {
	Id            int64           `gorm:"column:id;primary_key;AUTO_INCREMENT;NOT NULL"`
	Cid           string          `gorm:"uniqueIndex:idx_cid;column:cid;default:;NOT NULL"`
	OrderId       string          `gorm:"index:idx_order_id;column:order_id;default:;NOT NULL"`
	AccountId     string          `gorm:"index:idx_acc_id;column:account_id;default:;NOT NULL"`
	Account       string          `gorm:"column:account;default:;NOT NULL"`
	ManagerAid    int             `gorm:"column:manager_aid;default:0;NOT NULL"`
	ManagerSubAid int             `gorm:"column:manager_sub_aid;default:0;NOT NULL"`
	Manager       string          `gorm:"index:idx_manager;column:manager;default:;NOT NULL"`
	Name          string          `gorm:"column:name;default:;NOT NULL"`
	Note          string          `gorm:"column:note;default:;NOT NULL"`
	Price         decimal.Decimal `gorm:"price:amount; type:decimal(50,10) NOT NULL DEFAULT '0' COMMENT '';"`
	Num           int64           `gorm:"column:num;default:0;NOT NULL"`
	BeginAt       int64           `gorm:"column:begin_at;default:0;NOT NULL"`
	ExpiredAt     int64           `gorm:"column:expired_at;default:0;NOT NULL"`
	Status        int             `gorm:"column:status;default:0;NOT NULL"`
	CreatedAt     time.Time       `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt     time.Time       `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

func (t *CouponSetInfo) TableName() string {
	return "t_coupon_set_info"
}

func (t *CouponSetInfo) InitCid() {
	uid, _ := uuid.NewUUID()
	t.Cid = strings.ReplaceAll(uid.String(), "-", "")
}
