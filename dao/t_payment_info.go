package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) FindPaymentInfoByOrderId(orderID string) (list []tables.PaymentInfo, err error) {
	err = d.db.Where("order_id=? and pay_status=? and refund_status=? and cancel_status=?",
		orderID, tables.PayStatusSuccess, tables.RefundStatusDefault, tables.CancelStatusDefault).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetPaymentInfoByOrderId(orderID string) (payment tables.PaymentInfo, err error) {
	err = d.db.Where("order_id=? and pay_status=? and refund_status=? and cancel_status=?",
		orderID, tables.PayStatusSuccess, tables.RefundStatusDefault, tables.CancelStatusDefault).First(&payment).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
