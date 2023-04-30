package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) FindSubAccountAutoMintTotalProvider() (list []string, err error) {
	err = d.parserDb.Model(&tables.TableSubAccountAutoMintStatement{}).Select("service_provider_acc_id").Distinct("service_provider_acc_id").Group("service_provider_acc_id").Scan(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetSubAccountAutoMintByTxHash(txHash string) (list tables.TableSubAccountAutoMintStatement, err error) {
	err = d.parserDb.Where("tx_hash=?", txHash).First(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetLatestSubAccountAutoMintStatementByType(providerId string, txType tables.SubAccountAutoMintTxType) (list tables.TableSubAccountAutoMintStatement, err error) {
	err = d.parserDb.Where("service_provider_acc_id=? and tx_type=?", providerId, txType).Order("id desc").First(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindSubAccountAutoMintStatements(providerId string, txType tables.SubAccountAutoMintTxType, id uint64) (list []*tables.TableSubAccountAutoMintStatement, err error) {
	err = d.parserDb.Where("id>? and service_provider_acc_id=? and tx_type=?", id, providerId, txType).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
