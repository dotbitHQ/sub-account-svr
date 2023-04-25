package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DbDao) GetOrderByOrderID(orderID string) (order tables.OrderInfo, err error) {
	err = d.db.Where("order_id=?", orderID).First(&order).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindOrderByPayment(begin, end string, account ...string) (list []tables.OrderInfo, err error) {
	db := d.db.Model(&tables.OrderInfo{}).Where("pay_status=? AND pay_time>=? AND pay_time<?", tables.PayStatusPaid, begin, end)
	if len(account) > 0 && account[0] != "" {
		db = db.Where("account=?", account[0])
	}
	err = db.Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) UpdateAutoPaymentIdById(ids []uint64, paymentId string) error {
	if len(ids) <= 0 {
		return nil
	}
	return d.db.Model(&tables.OrderInfo{}).Where("id in (?)", ids).Updates(map[string]interface{}{
		"auto_payment_id": paymentId,
	}).Error
}

func (d *DbDao) UpdateOrderStatusOkWithSmtRecord(paymentInfo tables.PaymentInfo, smtRecord tables.TableSmtRecordInfo) (rowsAffected int64, e error) {
	e = d.db.Transaction(func(tx *gorm.DB) error {
		tmpTx := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND pay_status=?",
				paymentInfo.OrderId, tables.PayStatusUnpaid).
			Updates(map[string]interface{}{
				"pay_status": tables.PayStatusPaid,
			})

		if tmpTx.Error != nil {
			return tmpTx.Error
		}
		rowsAffected = tmpTx.RowsAffected
		log.Info("UpdateOrderStatusOkWithSmtRecord:", rowsAffected, paymentInfo.OrderId)

		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&paymentInfo).Error; err != nil {
			return err
		}

		if err := tx.Model(tables.PaymentInfo{}).
			Where("pay_hash=?", paymentInfo.PayHash).
			Updates(map[string]interface{}{
				"pay_hash_status": tables.PayHashStatusConfirmed,
				"timestamp":       paymentInfo.Timestamp,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(tables.PaymentInfo{}).
			Where("order_id=? AND pay_hash!=? AND pay_hash_status=?",
				paymentInfo.OrderId, paymentInfo.PayHash, tables.PayHashStatusPending).
			Updates(map[string]interface{}{
				"pay_hash_status": tables.PayHashStatusRejected,
			}).Error; err != nil {
			return err
		}

		if rowsAffected == 0 { // multi pay hash
			return nil
		}

		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&smtRecord).Error; err != nil {
			return err
		}
		return nil
	})
	return
}

// =========

func (d *DbDao) GetMintOrderInProgressByAccountIdWithoutAddr(accountId, addr string) (info tables.OrderInfo, err error) {
	timestamp := tables.GetEfficientOrderTimestamp()
	err = d.db.Where("account_id=? AND timestamp>=? AND pay_address!=? AND action_type=? AND pay_status=?",
		accountId, timestamp, addr, tables.ActionTypeMint, tables.PayStatusPaid).
		Order("id DESC").First(&info).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetMintOrderInProgressByAccountIdWithAddr(accountId, addr string) (info tables.OrderInfo, err error) {
	timestamp := tables.GetEfficientOrderTimestamp()
	err = d.db.Where("account_id=? AND timestamp>=? AND pay_address=? AND action_type=? AND pay_status=?",
		accountId, timestamp, addr, tables.ActionTypeMint, tables.PayStatusPaid).
		Order("id DESC").First(&info).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) CreateOrderInfo(info tables.OrderInfo) error {
	return d.db.Create(&info).Error
}
