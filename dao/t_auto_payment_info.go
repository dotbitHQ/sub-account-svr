package dao

import (
	"das_sub_account/tables"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func (d *DbDao) GetAutoPaymentAmount(accountId, tokenId string, paymentStatus tables.PaymentStatus) (amount decimal.Decimal, err error) {
	err = d.db.Model(&tables.AutoPaymentInfo{}).Select("IFNULL(sum(amount),0)").Where("account_id=? and token_id=? and payment_status=?", accountId, tokenId, paymentStatus).Scan(&amount).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
