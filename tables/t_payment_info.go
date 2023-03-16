package tables

import (
	"time"
)

type PayStatus int
type RefundStatus int
type CancelStatus int

const (
	PayStatusDefault PayStatus = iota
	PayStatusPending
	PayStatusSuccess
	PayStatusFail
	PayStatusExpire
)

const (
	RefundStatusDefault RefundStatus = iota
	RefundStatusPending
	RefundStatusSuccess
)

const (
	CancelStatusDefault CancelStatus = iota
	CancelStatusCancel
)

type PaymentInfo struct {
	Id           int64        `gorm:"column:id;type:bigint(20);primary_key;AUTO_INCREMENT" json:"id"`
	OrderId      string       `gorm:"column:order_id;index:idx_order_id;type:varchar(255);comment:订单号;NOT NULL" json:"order_id"`
	TokenId      string       `gorm:"column:token_id;type:varchar(255);comment:支付代币ID;NOT NULL" json:"token_id"`
	Address      string       `gorm:"column:address;type:varchar(255);comment:付款地址;NOT NULL" json:"address"`
	Amount       float64      `gorm:"column:amount;type:decimal(60) unsigned;default:0;comment:付款金额;NOT NULL" json:"amount"`
	PaymentTx    string       `gorm:"column:payment_tx;type:varchar(255);comment:支付交易;NOT NULL" json:"payment_tx"`
	BlockNumber  int64        `gorm:"column:block_number;type:bigint(20);default:0;comment:支付交易区块高度;NOT NULL" json:"block_number"`
	PayStatus    PayStatus    `gorm:"column:pay_status;type:smallint(6);default:0;comment:支付状态（0：未支付；1：支付确认中；2：支付成功；3：支付失败；4：支付过期）;NOT NULL" json:"pay_status"`
	RefundHash   string       `gorm:"column:refund_hash;type:varchar(255);comment:退款交易hash;NOT NULL" json:"refund_hash"`
	RefundStatus RefundStatus `gorm:"column:refund_status;type:smallint(6);default:0;comment:退款状态：（ 0：无需退款，1：退款中，2：退款完成）;NOT NULL" json:"refund_status"`
	CancelStatus CancelStatus `gorm:"column:cancel_status;type:smallint(6);default:0;comment:订单取消标志（0：未取消，1：取消）;NOT NULL" json:"cancel_status"`
	CreatedAt    time.Time    `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt    time.Time    `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *PaymentInfo) TableName() string {
	return "t_payment_info"
}
