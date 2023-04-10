package dao

import "das_sub_account/tables"

func (d *DbDao) CreateRuleWhitelist(rl tables.RuleWhitelist) error {
	return d.db.Create(rl).Error
}
