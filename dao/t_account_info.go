package dao

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
	"time"
)

func (d *DbDao) GetAccountInfoByAccountId(accountId string) (acc tables.TableAccountInfo, err error) {
	err = d.parserDb.Where(" account_id=? ", accountId).Find(&acc).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetSubAccountListByParentAccountId(parentAccountId string, chainType common.ChainType, address, keyword string, limit, offset int, category tables.Category, orderType tables.OrderType) (list []tables.TableAccountInfo, err error) {
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

	switch orderType {
	case tables.OrderTypeAccountAsc:
		db = db.Order("account")
	case tables.OrderTypeAccountDesc:
		db = db.Order("account desc")
	case tables.OrderTypeRegisterAtAsc:
		db = db.Order("registered_at")
	case tables.OrderTypeRegisterAtDesc:
		db = db.Order("registered_at desc")
	case tables.OrderTypeExpiredAtAsc:
		db = db.Order("expired_at")
	case tables.OrderTypeExpiredAtDesc:
		db = db.Order("expired_at desc")
	default:
		db = db.Order("account")
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error

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
		recycledAt := time.Now().Add(-time.Hour * 24 * 90).Unix()
		if config.Cfg.Server.Net != common.DasNetTypeMainNet {
			recycledAt = time.Now().Add(-time.Hour * 24 * 3).Unix()
		}
		db = db.Where("expired_at<=? AND expired_at>=?", expiredAt, recycledAt)
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

func (d *DbDao) GetAccountList(chainType common.ChainType, address string, limit, offset int, category tables.Category, keyword string) (list []tables.TableAccountInfo, err error) {
	//err = d.parserDb.Where(" owner_chain_type=? AND owner=? ", chainType, address).
	//	Or(" manager_chain_type=? AND manager=? ", chainType, address).
	//	Order("account").Limit(limit).Offset(offset).Find(&list).Error
	//return

	db := d.parserDb.Where("((owner_chain_type=? AND owner=?)OR(manager_chain_type=? AND manager=?))", chainType, address, chainType, address)
	db = db.Where("status!=?", tables.AccountStatusOnCross)

	switch category {
	//case tables.CategoryDefault:
	case tables.CategoryMainAccount:
		db = db.Where("parent_account_id=''")
	case tables.CategorySubAccount:
		db = db.Where("parent_account_id!=''")
	//case tables.CategoryOnSale:
	//	db = db.Where("status=?", tables.AccountStatusOnSale)
	//case tables.CategoryExpireSoon:
	//	expiredAt := time.Now().Unix()
	//	expiredAt30Days := time.Now().Add(time.Hour * 24 * 30).Unix()
	//	db = db.Where("expired_at>=? AND expired_at<=?", expiredAt, expiredAt30Days)
	//case tables.CategoryToBeRecycled:
	//	expiredAt := time.Now().Unix()
	//	recycledAt := time.Now().Add(-time.Hour * 24 * 90).Unix()
	//	if config.Cfg.Server.Net != common.DasNetTypeMainNet {
	//		recycledAt = time.Now().Add(-time.Hour * 24 * 3).Unix()
	//	}
	//	db = db.Where("expired_at<=? AND expired_at>=?", expiredAt, recycledAt)
	case tables.CategoryEnableSubAccount:
		db = db.Where("parent_account_id='' AND enable_sub_account=?", tables.AccountEnableStatusOn)
	}

	if keyword != "" {
		db = db.Where("account LIKE ?", keyword+"%")
	}

	err = db.Order("account").Limit(limit).Offset(offset).Find(&list).Error

	return
}

func (d *DbDao) GetAccountListTotal(chainType common.ChainType, address string, category tables.Category, keyword string) (count int64, err error) {
	//err = d.parserDb.Model(tables.TableAccountInfo{}).Where(" owner_chain_type=? AND owner=? ", chainType, address).
	//	Or(" manager_chain_type=? AND manager=? ", chainType, address).Count(&count).Error
	//return
	db := d.parserDb.Model(tables.TableAccountInfo{}).Where("((owner_chain_type=? AND owner=?)OR(manager_chain_type=? AND manager=?))", chainType, address, chainType, address)
	db = db.Where("status!=?", tables.AccountStatusOnCross)

	switch category {
	//case tables.CategoryDefault:
	case tables.CategoryMainAccount:
		db = db.Where("parent_account_id=''")
	case tables.CategorySubAccount:
		db = db.Where("parent_account_id!=''")
	//case tables.CategoryOnSale:
	//	db = db.Where("status=?", tables.AccountStatusOnSale)
	//case tables.CategoryExpireSoon:
	//	expiredAt := time.Now().Unix()
	//	expiredAt30Days := time.Now().Add(time.Hour * 24 * 30).Unix()
	//	db = db.Where("expired_at>=? AND expired_at<=?", expiredAt, expiredAt30Days)
	//case tables.CategoryToBeRecycled:
	//	expiredAt := time.Now().Unix()
	//	recycledAt := time.Now().Add(-time.Hour * 24 * 90).Unix()
	//	if config.Cfg.Server.Net != common.DasNetTypeMainNet {
	//		recycledAt = time.Now().Add(-time.Hour * 24 * 3).Unix()
	//	}
	//	db = db.Where("expired_at<=? AND expired_at>=?", expiredAt, recycledAt)
	case tables.CategoryEnableSubAccount:
		db = db.Where("parent_account_id='' AND enable_sub_account=?", tables.AccountEnableStatusOn)
	}

	if keyword != "" {
		db = db.Where("account LIKE ?", keyword+"%")
	}

	err = db.Count(&count).Error

	return
}

func (d *DbDao) GetAccountListByAccountIds(accountIds []string) (list []tables.TableAccountInfo, err error) {
	if len(accountIds) == 0 {
		return
	}
	err = d.parserDb.Where("account_id IN(?)", accountIds).Find(&list).Error
	return
}

func (d *DbDao) GetSubAccountNum(parentAccountId string) (num int64, err error) {
	err = d.parserDb.Model(&tables.TableAccountInfo{}).Where("parent_account_id=?", parentAccountId).Count(&num).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetSubAccountNumDistinct(parentAccountId string) (num int64, err error) {
	err = d.parserDb.Model(&tables.TableAccountInfo{}).Where("parent_account_id=?", parentAccountId).Distinct("owner").Count(&num).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}

func (d *DbDao) GetNeedToRecycleList(timestamp int64, recycleLimit int) (list []tables.TableAccountInfo, err error) {
	err = d.parserDb.Where("expired_at<? AND parent_account_id!=''", timestamp).Limit(recycleLimit).Find(&list).Error
	return
}
