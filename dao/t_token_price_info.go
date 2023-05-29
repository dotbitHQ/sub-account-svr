package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) GetTokenById(tokenID tables.TokenId) (token tables.TTokenPriceInfo, err error) {
	err = d.parserDb.Where("token_id=?", tokenID).First(&token).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindTokens() (tokens map[string]*tables.TTokenPriceInfo, err error) {
	list := make([]*tables.TTokenPriceInfo, 0)
	err = d.parserDb.Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	tokens = make(map[string]*tables.TTokenPriceInfo)
	for _, v := range list {
		tokens[string(v.TokenId)] = v
	}
	return
}
