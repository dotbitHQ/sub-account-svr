package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) GetRecordsByAccountId(accountId string) (list []tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id=?", accountId).Find(&list).Error
	return
}

func (d *DbDao) GetRecordsByAccountIdAndLabel(accountId, label string) (list []tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id=? and label=?", accountId, label).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
