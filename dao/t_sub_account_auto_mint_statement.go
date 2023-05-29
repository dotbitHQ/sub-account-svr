package dao

import (
	"das_sub_account/tables"
	"gorm.io/gorm"
)

func (d *DbDao) FindSubAccountAutoMintTotalProvider() (list []string, err error) {
	res := make([]*tables.TableSubAccountAutoMintStatement, 0)
	err = d.parserDb.Distinct("service_provider_id").Find(&res).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	for _, v := range res {
		list = append(list, v.ServiceProviderId)
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
	err = d.parserDb.Where("service_provider_id=? and tx_type=?", providerId, txType).Order("id desc").First(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) FindSubAccountAutoMintStatements(providerId string, txType tables.SubAccountAutoMintTxType, blockNumber uint64) (list []*tables.TableSubAccountAutoMintStatement, err error) {
	err = d.parserDb.Where("service_provider_id=? and tx_type=? and block_number > ?", providerId, txType, blockNumber).Find(&list).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
