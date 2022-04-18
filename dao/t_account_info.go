package dao

import (
	"das_sub_account/tables"
	"github.com/DeAccountSystems/das-lib/common"
)

func (d *DbDao) GetAccountInfoByAccountId(accountId string) (acc tables.TableAccountInfo, err error) {
	err = d.parserDb.Where(" account_id=? ", accountId).Find(&acc).Error
	return
}

func (d *DbDao) GetSubAccountListByParentAccountId(parentAccountId string, chainType common.ChainType, address string, limit, offset int) (list []tables.TableAccountInfo, err error) {
	if address != "" {
		err = d.parserDb.Where("parent_account_id=? AND ((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))",
			parentAccountId, chainType, address, chainType, address).
			Order("account").Limit(limit).Offset(offset).Find(&list).Error
	} else {
		err = d.parserDb.Where("parent_account_id=?", parentAccountId).
			Order("account").Limit(limit).Offset(offset).Find(&list).Error
	}
	return
}

func (d *DbDao) GetSubAccountListTotalByParentAccountId(parentAccountId string, chainType common.ChainType, address string) (count int64, err error) {
	if address != "" {
		err = d.parserDb.Where("parent_account_id=? AND ((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))",
			parentAccountId, chainType, address, chainType, address).
			Count(&count).Error
	} else {
		err = d.parserDb.Model(tables.TableAccountInfo{}).
			Where("parent_account_id=?", parentAccountId).Count(&count).Error
	}
	return
}

func (d *DbDao) GetAccountList(chainType common.ChainType, address string, limit, offset int) (list []tables.TableAccountInfo, err error) {
	err = d.parserDb.Where(" owner_chain_type=? AND owner=? ", chainType, address).
		Or(" manager_chain_type=? AND manager=? ", chainType, address).
		Order("account").Limit(limit).Offset(offset).Find(&list).Error
	return
}

func (d *DbDao) GetAccountListTotal(chainType common.ChainType, address string) (count int64, err error) {
	err = d.parserDb.Model(tables.TableAccountInfo{}).Where(" owner_chain_type=? AND owner=? ", chainType, address).
		Or(" manager_chain_type=? AND manager=? ", chainType, address).Count(&count).Error
	return
}

func (d *DbDao) GetAccountListByAccountIds(accountIds []string) (list []tables.TableAccountInfo, err error) {
	err = d.parserDb.Where("account_id IN(?)", accountIds).Find(&list).Error
	return
}
