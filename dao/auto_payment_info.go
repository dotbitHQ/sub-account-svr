package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) FindAutoPaymentInfo(parentAccountId string, page, size int) (resp []tables.AutoPaymentInfo, total int64, err error) {
	db := d.db.Model(&tables.AutoPaymentInfo{}).Where("account_id=?", parentAccountId).Order("id desc")
	if err = db.Count(&total).Error; err != nil && err != gorm.ErrRecordNotFound {
		return
	}
	err = db.Offset((page - 1) * size).Limit(size).Find(&resp).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
