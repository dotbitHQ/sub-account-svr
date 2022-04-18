package dao

import "das_sub_account/tables"

func (d *DbDao) GetSmtInfoBySubAccountIds(subAccountIds []string) (list []tables.TableSmtInfo, err error) {
	err = d.parserDb.Where("account_id IN(?)", subAccountIds).Find(&list).Error
	return
}
