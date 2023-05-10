package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) CreateRuleWhitelist(rl tables.RuleWhitelist) error {
	return d.db.Create(rl).Error
}

func (d *DbDao) GetRulesBySubAccountId(parentAccountId string, ruleType tables.RuleType, accountId string) (res tables.RuleWhitelist, err error) {
	err = d.db.Where("parent_account_id=? and rule_type=? and account_id=? and tx_status=?",
		parentAccountId, ruleType, accountId, tables.TxStatusCommitted).First(&res).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
