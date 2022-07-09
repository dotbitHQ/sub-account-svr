package dao

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

func (d *DbDao) GetAccountInfoByAccountId(accountId string) (acc tables.TableAccountInfo, err error) {
	err = d.parserDb.Where(" account_id=? ", accountId).Find(&acc).Error
	return
}

func (d *DbDao) GetSubAccountListByParentAccountId(parentAccountId string, chainType common.ChainType, address, keyword string, limit, offset int, category tables.Category) (list []tables.TableAccountInfo, err error) {
	db := d.parserDb.Where("parent_account_id=?", parentAccountId)
	if address != "" {
		db = db.Where("((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))", chainType, address, chainType, address)
	}
	switch category {
	//case tables.CategoryDefault:
	//case tables.CategoryMainAccount:
	//case tables.CategorySubAccount:
	//case tables.CategoryOnSale:
	case tables.CategoryExpireSoon:
		expiredAt := time.Now().Unix()
		expiredAt30Days := time.Now().Add(time.Hour * 24 * 30).Unix()
		db = db.Where("expired_at>=? AND expired_at<=?", expiredAt, expiredAt30Days)
	case tables.CategoryToBeRecycled:
		expiredAt := time.Now().Unix()
		recycledAt := time.Now().Add(-time.Hour * 24 * 90).Unix()
		if config.Cfg.Server.Net != common.DasNetTypeMainNet {
			recycledAt = time.Now().Add(-time.Hour * 24 * 3).Unix()
		}
		db = db.Where("expired_at<=? AND expired_at>=?", expiredAt, recycledAt)
	}
	if keyword != "" {
		db = db.Where("account LIKE ?", "%"+keyword+"%")
	}
	err = db.Order("account").Limit(limit).Offset(offset).Find(&list).Error

	//if address != "" {
	//	err = d.parserDb.Where("parent_account_id=? AND ((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))",
	//		parentAccountId, chainType, address, chainType, address).
	//		Order("account").Limit(limit).Offset(offset).Find(&list).Error
	//} else {
	//	err = d.parserDb.Where("parent_account_id=?", parentAccountId).
	//		Order("account").Limit(limit).Offset(offset).Find(&list).Error
	//}
	return
}

func (d *DbDao) GetSubAccountListTotalByParentAccountId(parentAccountId string, chainType common.ChainType, address, keyword string, category tables.Category) (count int64, err error) {
	db := d.parserDb.Model(tables.TableAccountInfo{}).Where("parent_account_id=?", parentAccountId)
	if address != "" {
		db = db.Where("((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))", chainType, address, chainType, address)
	}
	switch category {
	//case tables.CategoryDefault:
	//case tables.CategoryMainAccount:
	//case tables.CategorySubAccount:
	//case tables.CategoryOnSale:
	case tables.CategoryExpireSoon:
		expiredAt := time.Now().Unix()
		expiredAt30Days := time.Now().Add(time.Hour * 24 * 30).Unix()
		db = db.Where("expired_at>=? AND expired_at<=?", expiredAt, expiredAt30Days)
	case tables.CategoryToBeRecycled:
		expiredAt := time.Now().Unix()
		db = db.Where("expired_at<=?", expiredAt)
	}
	if keyword != "" {
		db = db.Where("account LIKE ?", "%"+keyword+"%")
	}
	err = db.Count(&count).Error

	return

	//if address != "" {
	//	err = d.parserDb.Where("parent_account_id=? AND ((owner_chain_type=? AND `owner`=?) OR (manager_chain_type=? AND manager=?))",
	//		parentAccountId, chainType, address, chainType, address).
	//		Count(&count).Error
	//} else {
	//	err = d.parserDb.Model(tables.TableAccountInfo{}).
	//		Where("parent_account_id=?", parentAccountId).Count(&count).Error
	//}
	//return
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
