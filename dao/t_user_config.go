package dao

import (
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
)

func (d *DbDao) UpdateMintConfig(account string, mintConfig *tables.MintConfig) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	return d.db.Select("account", "account_id", "mint_config").Where("account_id=?", accountId).Save(&tables.UserConfig{
		Account:    account,
		AccountId:  accountId,
		MintConfig: mintConfig,
	}).Error
}

func (d *DbDao) UpdatePaymentConfig(account string, paymentConfig *tables.PaymentConfig) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	return d.db.Select("account", "account_id", "payment_config").Where("account_id=?", accountId).Save(&tables.UserConfig{
		Account:       account,
		AccountId:     accountId,
		PaymentConfig: paymentConfig,
	}).Error
}

func (d *DbDao) GetUserPaymentConfig(accountId string) (paymentConfig tables.PaymentConfig, err error) {
	userCfg := &tables.UserConfig{}
	err = d.db.Where("account_id=?", accountId).First(userCfg).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
		return
	}
	if userCfg.PaymentConfig != nil && userCfg.PaymentConfig.CfgMap != nil {
		paymentConfig = *userCfg.PaymentConfig
	} else {
		paymentConfig.CfgMap = make(map[string]tables.PaymentConfigElement)
	}
	return
}
