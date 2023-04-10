package tables

import (
	"time"
)

//type PayStatus int
//type RefundStatus int
type CancelStatus int

//const (
//	PayStatusDefault PayStatus = iota
//	PayStatusPending
//	PayStatusSuccess
//	PayStatusFail
//	PayStatusExpire
//)

//const (
//	RefundStatusDefault RefundStatus = iota
//	RefundStatusPending
//	RefundStatusSuccess
//)

const (
	CancelStatusDefault CancelStatus = iota
	CancelStatusCancel
)

//type PaymentInfo struct {
//	Id           int64        `gorm:"column:id;AUTO_INCREMENT" json:"id"`
//	OrderId      string       `gorm:"column:order_id;index:idx_order_id;type:varchar(255);comment:订单号;NOT NULL" json:"order_id"`
//	TokenId      string       `gorm:"column:token_id;type:varchar(255);comment:支付代币ID;NOT NULL" json:"token_id"`
//	Address      string       `gorm:"column:address;type:varchar(255);comment:付款地址;NOT NULL" json:"address"`
//	USDPrice     float64      `gorm:"column:usd_price;type:decimal(50,2);default:0.00;comment:美元价格;NOT NULL" json:"usd_price"`
//	Amount       float64      `gorm:"column:amount;type:decimal(60) unsigned;default:0;comment:付款金额;NOT NULL" json:"amount"`
//	PaymentTx    string       `gorm:"column:payment_tx;type:varchar(255);comment:支付交易;NOT NULL" json:"payment_tx"`
//	BlockNumber  int64        `gorm:"column:block_number;type:bigint(20);default:0;comment:支付交易区块高度;NOT NULL" json:"block_number"`
//	PayStatus    PayStatus    `gorm:"column:pay_status;type:smallint(6);default:0;comment:支付状态（0：未支付；1：支付确认中；2：支付成功；3：支付失败；4：支付过期）;NOT NULL" json:"pay_status"`
//	RefundHash   string       `gorm:"column:refund_hash;type:varchar(255);comment:退款交易hash;NOT NULL" json:"refund_hash"`
//	RefundStatus RefundStatus `gorm:"column:refund_status;type:smallint(6);default:0;comment:退款状态：（ 0：无需退款，1：退款中，2：退款完成）;NOT NULL" json:"refund_status"`
//	CancelStatus CancelStatus `gorm:"column:cancel_status;type:smallint(6);default:0;comment:订单取消标志（0：未取消，1：取消）;NOT NULL" json:"cancel_status"`
//	CreatedAt    time.Time    `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
//	UpdatedAt    time.Time    `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
//}

type PaymentInfo struct {
	Id            uint64        `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	PayHash       string        `json:"pay_hash" gorm:"column:pay_hash; uniqueIndex:uk_pay_hash; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	OrderId       string        `json:"order_id" gorm:"column:order_id; index:idx_order_id; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
	PayAddress    string        `json:"pay_address" gorm:"column:pay_address; index:idx_pay_address; type:varchar(255) NOT NULL DEFAULT'' COMMENT '';"`
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
