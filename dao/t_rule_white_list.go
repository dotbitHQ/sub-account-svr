package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) CreateRuleWhitelist(rl tables.RuleWhitelist) error {
	return d.db.Create(rl).Error
}

func (d *DbDao) FindRulesBySubAccountIds(txHash string, parentAccountId string, ruleType tables.RuleType, ruleIndex int) (list []tables.RuleWhitelist, err error) {
	err = d.db.Where("tx_hash=? and parent_account_id=? and rule_type=? and rule_index=?",
		txHash, parentAccountId, ruleType, ruleIndex).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
