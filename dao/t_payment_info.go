package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
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

func (d *DbDao) CreatePaymentInfo(info tables.PaymentInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&info).Error
}

func (d *DbDao) GetUnRefundList() (list []tables.PaymentInfo, err error) {
	timestamp := time.Now().Add(-time.Hour * 24 * 7).Unix()
	err = d.db.Where("timestamp>=? AND pay_hash_status=? AND refund_status=?",
		timestamp, tables.PayHashStatusConfirmed, tables.RefundStatusUnRefund).Find(&list).Error
	return
}

func (d *DbDao) UpdateRefundStatusToRefundIng(ids []uint64) error {
	return d.db.Model(tables.PaymentInfo{}).
		Where("id IN(?) AND pay_hash_status=? AND refund_status=?",
			ids, tables.PayHashStatusConfirmed, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefunding,
		}).Error
}
