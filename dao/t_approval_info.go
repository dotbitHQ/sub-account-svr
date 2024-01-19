package dao

import (
	"das_sub_account/tables"
	"errors"
	"gorm.io/gorm"
)

func (d *DbDao) GetAccountPendingApproval(accountId string) (approval tables.ApprovalInfo, err error) {
	err = d.parserDb.Where("account_id=? and status=?", accountId, tables.ApprovalStatusEnable).Order("id desc").First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetPendingApprovalByAccIdAndPlatform(accountId, platform string) (approval tables.ApprovalInfo, err error) {
	err = d.parserDb.Where("account_id=? and platform=? and status=?", accountId, platform, tables.ApprovalStatusEnable).Order("id desc").First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetApprovalByAccIdAndOutPoint(accountId, outpoint string) (approval tables.ApprovalInfo, err error) {
	err = d.parserDb.Where("account_id=? and outpoint=?", accountId, outpoint).Order("id desc").First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}
