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
	err = d.db.Where("order_id=?", orderID).
		Order("timestamp DESC").Limit(1).
		Find(&payment).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) CreatePaymentInfo(info tables.PaymentInfo, setInfo tables.CouponSetInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&info).Error; err != nil {
			return err
		}
		if setInfo.Id > 0 {
			if err := tx.Model(&setInfo).Where("id=? and status=?", setInfo.Id, tables.CouponSetInfoStatusCreated).Updates(map[string]interface{}{
				"status": tables.CouponSetInfoStatusPaid,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) GetUnRefundList() (list []tables.PaymentInfo, err error) {
	timestamp := tables.GetPaymentInfoTimestamp()
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

func (d *DbDao) UpdateRefundStatusToRefunded(payHash, orderId, refundHash string) error {
	return d.db.Model(tables.PaymentInfo{}).
		Where("pay_hash=? AND order_id=? AND pay_hash_status=? AND refund_status=?",
			payHash, orderId, tables.PayHashStatusConfirmed, tables.RefundStatusRefunding).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefunded,
			"refund_hash":   refundHash,
		}).Error
}

func (d *DbDao) GetPayHashStatusPendingList() (list []tables.PaymentInfo, err error) {
	timestamp := tables.GetPaymentInfoTimestampBefore24h()
	err = d.db.Where("timestamp>=? AND pay_hash_status=?",
		timestamp, tables.PayHashStatusPending).Find(&list).Error
	return
}

func (d *DbDao) GetRefundStatusRefundingList() (list []tables.PaymentInfo, err error) {
	timestamp := tables.GetPaymentInfoTimestamp()
	err = d.db.Where("timestamp>=? AND pay_hash_status=? AND refund_status=?",
		timestamp, tables.PayHashStatusConfirmed, tables.RefundStatusRefunding).Find(&list).Error
	return
}

func (d *DbDao) UpdatePayHashStatusToFailByDispute(payHash, orderId string) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND pay_status=?",
				orderId, tables.PayStatusPaid).
			Updates(map[string]interface{}{
				"order_status": tables.OrderStatusFail,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(tables.PaymentInfo{}).
			Where("pay_hash=? AND order_id=? AND pay_hash_status=?",
				payHash, orderId, tables.PayHashStatusConfirmed).
			Updates(map[string]interface{}{
				"pay_hash_status": tables.PayHashStatusFailByDispute,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}
