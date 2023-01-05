package dao

import "das_sub_account/tables"

func (d *DbDao) GetSmtInfoBySubAccountIds(subAccountIds []string) (list []tables.TableSmtInfo, err error) {
	err = d.parserDb.Where("account_id IN(?)", subAccountIds).Find(&list).Error
	return
}

func (d *DbDao) GetSmtInfoGroups() (list []tables.TableSmtInfo, err error) {
	err = d.parserDb.Table(tables.TableNameSmtInfo).Select("parent_account_id").Group("parent_account_id").Scan(&list).Error
	return
}

func (d *DbDao) GetSmtInfoGroupsByAccountIds(parentAccountIds []string) (list []tables.TableSmtInfo, err error) {
	err = d.parserDb.Table(tables.TableNameSmtInfo).Select("parent_account_id").Group("parent_account_id").Where("parent_account_id IN(?)", parentAccountIds).Scan(&list).Error
	return
}

func (d *DbDao) GetSmtInfoByParentId(parentAccountId string) (list []tables.TableSmtInfo, err error) {
	err = d.parserDb.Where("parent_account_id=? ", parentAccountId).Select("account_id, leaf_data_hash").Find(&list).Error
	return
}
