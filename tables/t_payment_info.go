package tables

import (
	"time"
)

type PaymentInfo struct {
	Id            uint64        `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	PayHash       string        `json:"pay_hash" gorm:"column:pay_hash; uniqueIndex:uk_pay_hash; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	OrderId       string        `json:"order_id" gorm:"column:order_id; index:idx_order_id; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	PayHashStatus PayHashStatus `json:"pay_hash_status" gorm:"column:pay_hash_status; type:smallint(6) NOT NULL DEFAULT'0' COMMENT '0-pending 1-confirmed 2-rejected';"`
	RefundHash    string        `json:"refund_hash" gorm:"column:refund_hash; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	RefundStatus  RefundStatus  `json:"refund_status" gorm:"column:refund_status; type:smallint(6) NOT NULL DEFAULT'0' COMMENT '0-default 1-unrefund 2-refunding 3-refunded';"`
	Timestamp     int64         `json:"timestamp" gorm:"column:timestamp; index:idx_timestamp; type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt     time.Time     `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt     time.Time     `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

func (m *PaymentInfo) TableName() string {
	return "t_payment_info"
}

type PayHashStatus int

const (
	PayHashStatusPending   PayHashStatus = 0
	PayHashStatusConfirmed PayHashStatus = 1
	PayHashStatusRejected  PayHashStatus = 2
)

type RefundStatus int

const (
	RefundStatusDefault   RefundStatus = 0
	RefundStatusUnRefund  RefundStatus = 1
	RefundStatusRefunding RefundStatus = 2
	RefundStatusRefunded  RefundStatus = 3
)

func GetPaymentInfoTimestamp() int64 {
	return time.Now().Add(-time.Hour * 24 * 3).UnixMilli()
}
