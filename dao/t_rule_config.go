package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) CreateRuleConfig(priceConfig tables.RuleConfig) error {
	return d.parserDb.Create(priceConfig).Error
}

func (d *DbDao) GetRuleConfigByAccountId(accountId string) (ruleConfig tables.RuleConfig, err error) {
	err = d.parserDb.Where("account_id=?", accountId).First(&ruleConfig).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
