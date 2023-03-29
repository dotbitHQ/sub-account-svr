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

func (d *DbDao) GetRecordsByAccountIdAndTypeAndLabel(accountId, valueType, label string) (record tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id=? and type=? and label=?", accountId, valueType, label).Order("id desc").Limit(1).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
