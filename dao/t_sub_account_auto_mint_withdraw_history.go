package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) GetLatestSubAccountAutoMintWithdrawHistory(providerAccountId string) (a tables.TableSubAccountAutoMintWithdrawHistory, err error) {
	err = d.db.Where("service_provider_account_id = ?", providerAccountId).Order("id desc").First(&a).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindSubAccountAutoMintWithdrawHistoryByTaskId(taskId string) (list []*tables.TableSubAccountAutoMintWithdrawHistory, err error) {
	err = d.db.Where("task_id = ?", taskId).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
