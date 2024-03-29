package dao

import (
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DbDao) CreateUserConfigWithMintConfig(info tables.UserConfig, mintConfig tables.MintConfig) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&info).Error; err != nil {
			return err
		}
		if err := tx.Model(&tables.UserConfig{}).
			Where("account_id=?", info.AccountId).
			Updates(map[string]interface{}{
				"mint_config": &mintConfig,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) GetMintConfig(accountId string) (mintConfig tables.MintConfig, err error) {
	userCfg := &tables.UserConfig{}
	err = d.db.Where("account_id=?", accountId).First(userCfg).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
		return
	}
	if userCfg.MintConfig != nil {
		mintConfig = *userCfg.MintConfig
	}
	return
}

func (d *DbDao) CreateUserConfigWithPaymentConfig(info tables.UserConfig, paymentConfig tables.PaymentConfig) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&info).Error; err != nil {
			return err
		}
		if err := tx.Model(&tables.UserConfig{}).
			Where("account_id=?", info.AccountId).
			Updates(map[string]interface{}{
				"payment_config": &paymentConfig,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) UpdatePaymentConfig(account string, paymentConfig *tables.PaymentConfig) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	return d.db.Model(&tables.UserConfig{}).Where("account_id=?", accountId).Updates(map[string]interface{}{
		"payment_config": paymentConfig,
	}).Error
}

func (d *DbDao) GetUserPaymentConfig(accountId string) (paymentConfig tables.PaymentConfig, err error) {
	paymentConfig.CfgMap = make(map[string]tables.PaymentConfigElement)

	userCfg := &tables.UserConfig{}
	err = d.db.Where("account_id=?", accountId).First(userCfg).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
		return
	}
	if userCfg.PaymentConfig != nil && userCfg.PaymentConfig.CfgMap != nil {
		paymentConfig = *userCfg.PaymentConfig
	}
	return
}
