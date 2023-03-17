package dao

import (
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
)

func (d *DbDao) UpdateMintConfig(account string, mintConfig tables.MintConfig) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	return d.db.Select("account", "account_id", "mint_config").Where("account_id=?", accountId).Save(&tables.UserConfig{
		Account:    account,
		AccountId:  accountId,
		MintConfig: mintConfig,
	}).Error
}

func (d *DbDao) UpdatePaymentConfig(account string, paymentConfig tables.PaymentConfig) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	return d.db.Select("account", "account_id", "payment_config").Where("account_id=?", accountId).Save(&tables.UserConfig{
		Account:       account,
		AccountId:     accountId,
		PaymentConfig: paymentConfig,
	}).Error
}
