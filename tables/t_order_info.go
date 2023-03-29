package tables

import (
	"time"
)

type OrderStatus int

const (
	OrderStatusDefault OrderStatus = 0
	OrderStatusPending OrderStatus = 1
	OrderStatusSuccess OrderStatus = 2
	OrderStatusFail    OrderStatus = 3
	OrderStatusCancel  OrderStatus = 4
)

type OrderInfo struct {
	Id              int64     `gorm:"column:id;AUTO_INCREMENT" json:"id"`
	OrderId         string    `gorm:"column:order_id;type:varchar(255);comment:订单号;NOT NULL" json:"order_id"`
	Account         string    `gorm:"column:account;type:varchar(255);comment:账号名;NOT NULL" json:"account"`
	AccountId       string    `gorm:"column:account_id;index:idx_account_id;type:varchar(255);comment:账号id;NOT NULL" json:"account_id"`
	Address         string    `gorm:"column:address;type:varchar(255);comment:下单地址;NOT NULL" json:"address"`
	ParentAccount   string    `gorm:"column:parent_account;type:varchar(255);comment:父账号;NOT NULL" json:"parent_account"`
	ParentAccountId string    `gorm:"column:parent_account_id;index:idx_p_account_id;type:varchar(255);comment:父账号id;NOT NULL" json:"parent_account_id"`
	Years           uint      `gorm:"column:years;type:int(10) unsigned;default:0;comment:购买多少年;NOT NULL" json:"years"`
	RegisteredAt    uint64    `gorm:"column:registered_at;type:bigint(20) unsigned;default:0;comment:注册成功时间;NOT NULL" json:"registered_at"`
	OrderStatus     int       `gorm:"column:order_status;type:smallint(6);default:0;comment:订单状态（0：未支付，1：支付中，2：支付成功；3：支付失败，4：取消）;NOT NULL" json:"order_status"`
	AutoPaymentId   string    `gorm:"column:auto_payment_id;type:varchar(255);comment:关联的自动打款id;NOT NULL" json:"auto_payment_id"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *OrderInfo) TableName() string {
	return "t_order_info"
}
