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

func (d *DbDao) GetRecordsByAccountIdAndTypeAndLabel(accountId, valueType, label string, keys []string) (record tables.TableRecordsInfo, err error) {
	err = d.parserDb.Where("account_id=? and type=? and label=? and `key` in (?)", accountId, valueType, label, keys).Order("id desc").First(&record).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
