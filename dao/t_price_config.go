package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) CreatePriceConfig(priceConfig tables.PriceConfig) error {
	return d.db.Create(priceConfig).Error
}

func (d *DbDao) GetPriceConfigByTxHash(txHash string) (priceConfig tables.PriceConfig, err error) {
	err = d.db.Where("tx_hash=?", txHash).First(&priceConfig).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) UpdatePriceConfigByTxHash(txHash string) (err error) {
	err = d.db.Where("tx_hash=?", txHash).Updates(map[string]interface{}{
		"tx_status": tables.TxStatusPending,
	}).Error
	return
}
