package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) CreateRuleWhitelist(rl tables.RuleWhitelist) error {
	return d.db.Create(rl).Error
}

func (d *DbDao) GetRulesBySubAccountIds(parentAccountId string, ruleType tables.RuleType, accountIds []string) (list []*tables.RuleWhitelist, err error) {
	err = d.db.Where("parent_account_id=? and rule_type=? and account_id in (?) and tx_status=?",
		parentAccountId, ruleType, accountIds, tables.TxStatusCommitted).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
