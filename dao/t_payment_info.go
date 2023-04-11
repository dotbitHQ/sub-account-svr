package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DbDao) FindPaymentInfoByOrderId(orderID string) (list []tables.PaymentInfo, err error) {
	err = d.db.Where("order_id=? and pay_hash_status=? and refund_status=?",
		orderID, tables.PayHashStatusConfirmed, tables.RefundStatusDefault).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetPaymentInfoByOrderId(orderID string) (payment tables.PaymentInfo, err error) {
	err = d.db.Where("order_id=? and pay_hash_status=? and refund_status=?",
		orderID, tables.PayHashStatusConfirmed, tables.RefundStatusDefault).First(&payment).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

// =====
func (d *DbDao) CreatePaymentInfo(info tables.PaymentInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&info).Error
}
