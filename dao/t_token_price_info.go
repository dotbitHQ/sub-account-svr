package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) GetTokenById(tokenID string) (token tables.TTokenPriceInfo, err error) {
	err = d.parserDb.Where("token_id=?", tokenID).First(&token).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
