package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

type OrderAndPaymentInfo struct {
	OrderInfo   tables.OrderInfo   `gorm:"embedded" json:"order_info"`
	PaymentInfo tables.PaymentInfo `gorm:"embedded" json:"payment_info"`
}

func (d *DbDao) GetOrderByOrderID(orderID string) (order tables.OrderInfo, err error) {
	err = d.db.Where("order_id=?", orderID).First(&order).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindOrderPaymentInfo(begin, end string, account ...string) (list []OrderAndPaymentInfo, err error) {
	db := d.db.Table("t_order_info").Joins("left join t_payment_info on t_order_info.order_id=t_payment_info.order_id").
		Select("t_order_info.*,t_payment_info.*").Distinct("t_order_info.order_id").
		Where("t_payment_info.pay_status=? and t_payment_info.refund_status=? and cancel_status=? and t_order_info.created_at>=? and t_order_info.created_at<=?",
			tables.PaymentStatusSuccess, tables.RefundStatusDefault, tables.CancelStatusDefault, begin, end)
	if len(account) > 0 {
		db = db.Where("t_order_info.account=?", account[0])
	}
	err = db.Group("t_order_info.order_id").Scan(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
