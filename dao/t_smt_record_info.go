package dao

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
)

func (d *DbDao) CreateSmtRecordInfo(record tables.TableSmtRecordInfo) error {
	return d.db.Create(&record).Error
}

func (d *DbDao) CreateSmtRecordList(recordList []tables.TableSmtRecordInfo) error {
	return d.db.Create(&recordList).Error
}

func (d *DbDao) GetNeedDoDistributionRecordListNew(svrName string, action common.DasAction) (list []tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("task_id='' AND action=? AND svr_name=?", action, svrName).
		Order("parent_account_id,id").Limit(2000).Find(&list).Error
	return
}

func (d *DbDao) UpdateTaskDistribution(taskList []tables.TableTaskInfo, idsList [][]uint64) error {
	if len(taskList) == 0 {
		return nil
	}
	if len(taskList) != len(idsList) {
		return fmt.Errorf("len diff [%d] [%d]", len(taskList), len(idsList))
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&taskList).Error; err != nil {
			return err
		}
		for i, _ := range taskList {
			if err := tx.Model(tables.TableSmtRecordInfo{}).
				Where("id IN(?)", idsList[i]).
				Updates(map[string]interface{}{
					"task_id": taskList[i].TaskId,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DbDao) GetSmtRecordListByTaskId(taskId string) (list []tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("task_id=?", taskId).Find(&list).Error
	return
}

func (d *DbDao) GetChainSmtRecordListByTaskId(taskId string) (list []tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("task_id=? AND record_type=?", taskId, tables.RecordTypeChain).Find(&list).Error
	return
}

func (d *DbDao) GetSmtRecordListByTaskIds(taskIds []string, recordType tables.RecordType) (list []tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("task_id IN(?) AND record_type=?", taskIds, recordType).Find(&list).Error
	return
}

func (d *DbDao) GetSelfSmtRecordListByAccountIds(accountIds []string) (list []tables.TableSmtRecordInfo, err error) {
	if len(accountIds) == 0 {
		return
	}
	err = d.db.Where("account_id IN(?) AND record_type=?",
		accountIds, tables.RecordTypeDefault).Find(&list).Error
	return
}

func (d *DbDao) GetLatestSmtRecordByParentAccountIdAction(accountId, action, subAction string) (record tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("parent_account_id=? AND action=? AND sub_action=? AND record_type=?",
		accountId, action, subAction, tables.RecordTypeDefault).
		Order("nonce DESC").Limit(1).Find(&record).Error
	return
}

func (d *DbDao) GetLatestSmtRecordByAccountIdAction(accountId, action, subAction string) (record tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("account_id=? AND action=? AND sub_action=? AND record_type=?",
		accountId, action, subAction, tables.RecordTypeDefault).
		Order("nonce DESC").Limit(1).Find(&record).Error
	return
}

func (d *DbDao) GetLatestMintRecord(accountId, action, subAction string) (record tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("account_id=? AND action=? AND sub_action=?", accountId, action, subAction).
		Order("id DESC").Limit(1).Find(&record).Error
	return
}

func (d *DbDao) GetLatestNonceSmtRecordByAccountId(accountId string, recordType tables.RecordType) (record tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("account_id=? AND record_type=?", accountId, recordType).
		Order("nonce DESC").Limit(1).Find(&record).Error
	return
}

func (d *DbDao) UpdateRecordsToClosed(taskId string, diffNonceList []tables.TableSmtRecordInfo, closed bool) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if closed {
			if err := tx.Model(tables.TableTaskInfo{}).
				Where("task_id=?", taskId).
				Updates(map[string]interface{}{
					"task_type": tables.TaskTypeClosed,
				}).Error; err != nil {
				return err
			}
		}
		var ids []uint64
		for _, v := range diffNonceList {
			if err := tx.Where("account_id=? AND nonce=? AND record_type=? AND task_id!=?",
				v.AccountId, v.Nonce, tables.RecordTypeClosed, taskId).
				Delete(&tables.TableSmtRecordInfo{}).Error; err != nil {
				return err
			}
			ids = append(ids, v.Id)
		}
		if err := tx.Model(tables.TableSmtRecordInfo{}).
			Where("id IN(?)", ids).
			Updates(map[string]interface{}{
				"record_type": tables.RecordTypeClosed,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateMinSignInfo(mintSignInfo tables.TableMintSignInfo, list []tables.TableSmtRecordInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&mintSignInfo).Error; err != nil {
			return err
		}
		if err := tx.Create(&list).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) FindSmtRecordInfoByMintType(parentAccountId string, mintType tables.MintType, actions []string) (resp []tables.TableSmtRecordInfo, err error) {
	err = d.db.Where("parent_account_id=? and mint_type=? and action in (?)", parentAccountId, mintType, actions).Find(&resp).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return
}
