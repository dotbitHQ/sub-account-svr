package dao

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DbDao) GetOrderByOrderID(orderID string) (order tables.OrderInfo, err error) {
	err = d.db.Where("order_id=?", orderID).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
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

func (d *DbDao) UpdateOrderPayStatusOkWithCoupon(paymentInfo tables.PaymentInfo, setInfo tables.CouponSetInfo, coupons []tables.CouponInfo) (rowsAffected int64, e error) {
	e = d.db.Transaction(func(tx *gorm.DB) error {
		tmpTx := tx.Model(tables.OrderInfo{}).
			Where("order_id=? AND pay_status=?",
				paymentInfo.OrderId, tables.PayStatusUnpaid).
			Updates(map[string]interface{}{
				"pay_status":   tables.PayStatusPaid,
				"order_status": tables.OrderStatusSuccess,
			})

		if tmpTx.Error != nil {
			return tmpTx.Error
		}
		rowsAffected = tmpTx.RowsAffected
		log.Info("UpdateOrderPayStatusOkWithCoupon:", rowsAffected, paymentInfo.OrderId)

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

		setInfo.Status = tables.CouponSetInfoStatusSuccess
		if err := tx.Save(&setInfo).Error; err != nil {
			return err
		}

		if err := tx.Create(&coupons).Error; err != nil {
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

func (d *DbDao) CreateOrderInfo(info tables.OrderInfo, paymentInfo tables.PaymentInfo, setInfo tables.CouponSetInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&info).Error; err != nil {
			return err
		}
		if paymentInfo.PayHash != "" {
			if err := tx.Create(&paymentInfo).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&setInfo).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateOrderInfoWithCoupon(info tables.OrderInfo, paymentInfo tables.PaymentInfo, couponInfo tables.CouponInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&info).Error; err != nil {
			return err
		}
		if paymentInfo.PayHash != "" {
			if err := tx.Create(&paymentInfo).Error; err != nil {
				return err
			}
		}
		if couponInfo.Id > 0 {
			couponInfo.Status = tables.CouponStatusUsed
			if err := tx.Where("status = ?", tables.CouponStatusNormal).Save(&couponInfo).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) GetOrderAmount(accountId string, paid bool) (result map[string]decimal.Decimal, err error) {
	list := make([]*tables.OrderInfo, 0)
	db := d.db.Where("parent_account_id=? and pay_status=? and order_status=? and action_type in (?)",
		accountId, tables.PayStatusPaid, tables.OrderStatusSuccess, []tables.ActionType{tables.ActionTypeMint, tables.ActionTypeRenew})
	if paid {
		db = db.Where("auto_payment_id != ''")
	}
	err = db.Find(&list).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}

	tokens, err := d.FindTokens()
	if err != nil {
		return nil, err
	}

	feeRate := decimal.NewFromFloat(0.85)
	minPriceFee := decimal.NewFromFloat(0.99).
		Add(decimal.NewFromFloat(config.Cfg.Das.AutoMint.ServiceFeeMin))

	result = make(map[string]decimal.Decimal)
	for _, v := range list {
		amount := decimal.Zero
		if v.TokenId != "" && v.Amount.GreaterThan(decimal.Zero) {
			token := tokens[v.TokenId]
			couponMinPrice := minPriceFee.Div(decimal.NewFromFloat(0.15)).Mul(decimal.NewFromInt(int64(v.Years)))
			tokenMinPrice := couponMinPrice.Mul(decimal.New(1, token.Decimals)).DivRound(token.Price, token.Decimals)
			fee := minPriceFee.Mul(decimal.NewFromInt(int64(v.Years))).Mul(decimal.New(1, token.Decimals)).DivRound(token.Price, token.Decimals)
			amount = v.Amount.Sub(v.PremiumAmount)
			if v.CouponCode == "" {
				if v.USDAmount.GreaterThan(decimal.Zero) {
					if v.USDAmount.GreaterThan(couponMinPrice) {
						amount = amount.Mul(feeRate)
					} else {
						amount = amount.Sub(fee)
					}
				} else {
					if v.Amount.GreaterThan(tokenMinPrice) {
						amount = amount.Mul(feeRate)
					} else {
						amount = amount.Sub(fee)
					}
				}
			} else {
				couponSetInfo, err := d.GetSetInfoByCoupon(v.CouponCode)
				if err != nil {
					return nil, err
				}
				if v.USDAmount.GreaterThan(couponSetInfo.Price) {
					amount = v.USDAmount.Sub(couponSetInfo.Price).Mul(decimal.New(1, token.Decimals)).DivRound(token.Price, token.Decimals)
					amount = amount.Mul(feeRate)
				}
			}
		}
		result[v.TokenId] = result[v.TokenId].Add(amount)
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

func (d *DbDao) GetPendingOrderByAccIdAndActionType(accountId string, actionType tables.ActionType) (order tables.OrderInfo, err error) {
	err = d.db.Where("account_id=? and action_type=? and order_status=?", accountId, actionType, tables.OrderStatusDefault).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetOrderByCoupon(coupon string) (order tables.OrderInfo, err error) {
	err = d.db.Where("coupon_code=?", coupon).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetOrderAmountByAccIdAndTokenId(accountId string, tokenId tables.TokenId) (amount decimal.Decimal, err error) {
	order := &tables.OrderInfo{}
	err = d.db.Model(order).Select("sum(amount) as amount").
		Where("(account_id=? or parent_account_id=?) and token_id=? and pay_status=? and order_status=?",
			accountId, accountId, tokenId, tables.PayStatusPaid, tables.OrderStatusSuccess).First(order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	amount = order.Amount
	return
}
