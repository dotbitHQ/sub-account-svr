package dao

import (
	"das_sub_account/tables"
	"github.com/shopspring/decimal"
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

func (d *DbDao) FindOrderByPayment(end int64, accountId string) (list []*tables.OrderInfo, err error) {
	db := d.db.Model(&tables.OrderInfo{}).Where("auto_payment_id = '' AND pay_status=? AND order_status=? AND timestamp<?", tables.PayStatusPaid, tables.OrderStatusSuccess, end)
	if accountId != "" {
		db = db.Where("parent_account_id=?", accountId)
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

func (d *DbDao) UpdateOrderPayStatusOkWithSmtRecord(paymentInfo tables.PaymentInfo, smtRecord tables.TableSmtRecordInfo) (rowsAffected int64, e error) {
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

func (d *DbDao) GetMintOrderInProgressByAccountIdWithoutAddr(accountId, addr string, actionType tables.ActionType) (info tables.OrderInfo, err error) {
	timestamp := tables.GetEfficientOrderTimestamp()
	err = d.db.Where("account_id=? AND timestamp>=? AND pay_address!=? AND action_type=? AND pay_status=?",
		accountId, timestamp, addr, actionType, tables.PayStatusPaid).
		Order("id DESC").First(&info).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetMintOrderInProgressByAccountIdWithAddr(accountId, addr string, actionType tables.ActionType) (info tables.OrderInfo, err error) {
	timestamp := tables.GetEfficientOrderTimestamp()
	err = d.db.Where("account_id=? AND timestamp>=? AND pay_address=? AND action_type=? AND pay_status=?",
		accountId, timestamp, addr, actionType, tables.PayStatusPaid).
		Order("id DESC").First(&info).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) CreateOrderInfo(info tables.OrderInfo, paymentInfo tables.PaymentInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&info).Error; err != nil {
			return err
		}
		if paymentInfo.PayHash != "" {
			if err := tx.Create(&paymentInfo).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type OrderAmountInfo struct {
	TokenId string          `json:"token_id" gorm:"column:token_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Amount  decimal.Decimal `json:"amount" gorm:"column:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
}

func (d *DbDao) GetOrderAmount(accountId string, paid bool) (result map[string]decimal.Decimal, err error) {
	list := make([]*OrderAmountInfo, 0)
	db := d.db.Model(&tables.OrderInfo{}).Select("token_id, sum(amount-premium_amount) as amount").
		Where("parent_account_id=? and pay_status=? and order_status=?", accountId, tables.PayStatusPaid, tables.OrderStatusSuccess)
	if paid {
		db = db.Where("auto_payment_id != ''")
	}
	err = db.Group("token_id").Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}

	result = make(map[string]decimal.Decimal)
	for _, v := range list {
		result[v.TokenId] = result[v.TokenId].Add(v.Amount)
	}
	return
}

func (d *DbDao) GetNeedCheckOrderList() (list []tables.OrderInfo, err error) {
	timestamp := tables.GetEfficientOrderTimestamp()
	err = d.db.Where("timestamp>=? AND pay_status=? AND order_status=?",
		timestamp, tables.PayStatusPaid, tables.OrderStatusDefault).
		Order("timestamp").Limit(100).Find(&list).Error
	return
}

func (d *DbDao) UpdateOrderStatusForCheckMint(orderId string, oldStatus, newStatus tables.OrderStatus) error {
	return d.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND order_status=?", orderId, oldStatus).
			Updates(map[string]interface{}{
				"order_status": newStatus,
			}).Error; err != nil {
			return err
		}
		if newStatus == tables.OrderStatusFail {
			if err := tx.Model(tables.PaymentInfo{}).
				Where("order_id=? AND pay_hash_status=? AND refund_status=?",
					orderId, tables.PayHashStatusConfirmed, tables.RefundStatusDefault).
				Updates(map[string]interface{}{
					"refund_status": tables.RefundStatusUnRefund,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) UpdateOrderStatusForCheckRenew(orderId string, oldStatus, newStatus tables.OrderStatus) error {
	return d.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND order_status=?", orderId, oldStatus).
			Updates(map[string]interface{}{
				"order_status": newStatus,
			}).Error; err != nil {
			return err
		}
		if newStatus == tables.OrderStatusFail {
			if err := tx.Model(tables.PaymentInfo{}).
				Where("order_id=? AND pay_hash_status=? AND refund_status=?",
					orderId, tables.PayHashStatusConfirmed, tables.RefundStatusDefault).
				Updates(map[string]interface{}{
					"refund_status": tables.RefundStatusUnRefund,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) UpdateOrderStatusToFailForUnconfirmedPayHash(orderId, payHash string) error {
	return d.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND order_status=?", orderId, tables.OrderStatusDefault).
			Updates(map[string]interface{}{
				"order_status": tables.OrderStatusFail,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(tables.PaymentInfo{}).
			Where("order_id=? AND pay_hash=? AND pay_hash_status=? AND refund_status=?",
				orderId, payHash, tables.PayHashStatusPending, tables.RefundStatusDefault).
			Updates(map[string]interface{}{
				"pay_hash_status": tables.PayHashStatusRejected,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}
