package tables

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type PayStatus int

const (
	PayStatusUnpaid PayStatus = 0
	PayStatusPaid   PayStatus = 1
)

type OrderStatus int

const (
	OrderStatusDefault OrderStatus = 0
	OrderStatusSuccess OrderStatus = 1
	OrderStatusFail    OrderStatus = 2
)

type ActionType int

const (
	ActionTypeMint         ActionType = 0
	ActionTypeRenew        ActionType = 1
	ActionTypeCouponCreate ActionType = 2
)

type OrderInfo struct {
	Id                uint64                `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	OrderId           string                `json:"order_id" gorm:"column:order_id; uniqueIndex:uk_order_id; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	ActionType        ActionType            `json:"action_type" gorm:"column:action_type; type:smallint(6) NOT NULL DEFAULT'0' COMMENT '0-mint 1-renew';"`
	Account           string                `json:"account" gorm:"column:account; type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' COMMENT '';"`
	AccountId         string                `json:"account_id" gorm:"column:account_id; index:idx_account_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	ParentAccountId   string                `json:"parent_account_id" gorm:"column:parent_account_id; index:k_parent_account_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Years             uint64                `json:"years" gorm:"column:years; type:int(10) unsigned NOT NULL DEFAULT '0' COMMENT '';"`
	AlgorithmId       common.DasAlgorithmId `json:"algorithm_id" gorm:"column:algorithm_id; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE';"`
	PayAddress        string                `json:"pay_address" gorm:"column:pay_address; index:idx_pay_address; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	TokenId           string                `json:"token_id" gorm:"column:token_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CouponCode        string                `json:"coupon_code" gorm:"column:coupon_code; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CouponAmount      decimal.Decimal       `json:"coupon_amount" gorm:"column:coupon_amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	Amount            decimal.Decimal       `json:"amount" gorm:"column:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	USDAmount         decimal.Decimal       `json:"usd_amount" gorm:"column:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	PayStatus         PayStatus             `json:"pay_status" gorm:"column:pay_status; type:smallint(6) NOT NULL DEFAULT'0' COMMENT '0-unpaid 1-paid';"`
	OrderStatus       OrderStatus           `json:"order_status" gorm:"column:order_status; type:smallint(6) NOT NULL DEFAULT'0' COMMENT '0-default 1-cancel';"`
	Timestamp         int64                 `json:"timestamp" gorm:"column:timestamp; index:idx_timestamp; type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '';"`
	AutoPaymentId     string                `json:"auto_payment_id" gorm:"column:auto_payment_id; index:idx_auto_payment_id; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	SvrName           string                `json:"svr_name" gorm:"column:svr_name; index:k_svr_name; type:varchar(255) NOT NULL DEFAULT '' COMMENT 'smt tree';"`
	PremiumPercentage decimal.Decimal       `json:"premium_percentage" gorm:"column:premium_percentage; type:decimal(20,10) NOT NULL DEFAULT '0' COMMENT '';"`
	PremiumBase       decimal.Decimal       `json:"premium_base" gorm:"column:premium_base; type:decimal(20,10) NOT NULL DEFAULT '0' COMMENT '';"`
	PremiumAmount     decimal.Decimal       `json:"premium_amount" gorm:"column:premium_amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	MetaData          string                `json:"meta_data" gorm:"column:meta_data; type:text NOT NULL COMMENT '';"`
	CreatedAt         time.Time             `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt         time.Time             `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

func (t *OrderInfo) TableName() string {
	return "t_order_info"
}

func GetEfficientOrderTimestamp() int64 {
	return time.Now().Add(-time.Hour * 24 * 3).UnixMilli()
}

func GetParentAccountId(subAcc string) string {
	indexDot := strings.Index(subAcc, ".")
	parentAccountName := subAcc[indexDot+1:]
	log.Info("GetParentAccountId:", subAcc, parentAccountName)
	return common.Bytes2Hex(common.GetAccountIdByAccount(parentAccountName))
}
