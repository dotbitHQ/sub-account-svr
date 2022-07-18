package dao

import "das_sub_account/tables"

func (d *DbDao) GetCustomScriptInfo(accountId string) (info tables.TableCustomScriptInfo, err error) {
	err = d.parserDb.Where("account_id=?", accountId).Find(&info).Error
	return
}
