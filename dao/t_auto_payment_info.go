package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) GetAutoPaymentAmount(accountId, tokenId string, paymentStatus tables.PaymentStatus) (amount float64, err error) {
	err = d.db.Model(&tables.AutoPaymentInfo{}).Select("sum(amount)").Where("account_id=? and token_id=? and payment_status=?", accountId, tokenId, paymentStatus).Scan(&amount).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
