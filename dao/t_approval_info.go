package dao

import (
	"das_sub_account/tables"
	"errors"
	"gorm.io/gorm"
)

func (d *DbDao) CreateAccountApproval(info tables.ApprovalInfo) (err error) {
	err = d.db.Create(info).Error
	return
}

func (d *DbDao) UpdateAccountApproval(id uint64, info map[string]interface{}) (err error) {
	err = d.db.Model(&tables.ApprovalInfo{}).Where("id=?", id).Updates(info).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetAccountApprovalByOutpoint(outpoint string) (approval tables.ApprovalInfo, err error) {
	err = d.db.Where("outpoint=?", outpoint).First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetAccountApprovalByAccountId(accountId string) (approval tables.ApprovalInfo, err error) {
	err = d.db.Where("account_id=? and status=?", accountId, tables.ApprovalStatusEnable).First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

func (d *DbDao) GetAccountApprovalByAccountIdAndPlatform(accountId, platform string) (approval tables.ApprovalInfo, err error) {
	err = d.db.Where("account_id=? and platform=? and status=?", accountId, platform, tables.ApprovalStatusEnable).First(&approval).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}
