package dao

import (
	"das_sub_account/tables"
	"errors"
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	tokens = make(map[string]*tables.TTokenPriceInfo)
	for _, v := range list {
		tokens[string(v.TokenId)] = v
	}
	return
}

func (d *DbDao) GetTokenPriceList() (list []tables.TTokenPriceInfo, err error) {
	err = d.parserDb.Where("token_id NOT IN('bsc_bep20_usdt','eth_erc20_usdt','tron_trc20_usdt')").Order("id DESC").Find(&list).Error
	return
}
