package dao

import "das_sub_account/tables"

func (d *DbDao) GetRecordsByAccountId(accountId string) (list []tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id=?", accountId).Find(&list).Error
	return
}

func (d *DbDao) GetRecordsByAccountIds(accountIds []string) (list []tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id IN(?)", accountIds).Find(&list).Error
	return
}
