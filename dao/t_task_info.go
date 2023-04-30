package dao

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *DbDao) GetNeedDoCheckTxTaskList(svrName string) (list []tables.TableTaskInfo, err error) {
	err = d.db.Where("smt_status=? AND tx_status=? AND svr_name=?",
		tables.SmtStatusWriteComplete, tables.TxStatusPending, svrName).
		Find(&list).Error
	return
}

func (d *DbDao) UpdateTaskStatusToRejected(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return d.db.Model(tables.TableTaskInfo{}).
		Where("id IN(?) AND smt_status=? AND tx_status=?", ids, tables.SmtStatusWriteComplete, tables.TxStatusPending).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusNeedToRollback,
			"tx_status":  tables.TxStatusRejected,
		}).Error
}

func (d *DbDao) UpdateTaskTxStatusToPending(taskId string) error {
	return d.db.Model(tables.TableTaskInfo{}).Where("task_id=? AND smt_status=? AND tx_status=?",
		taskId, tables.SmtStatusWriting, tables.TxStatusUnSend).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusWriteComplete,
			"tx_status":  tables.TxStatusPending,
		}).Error
}

func (d *DbDao) GetNeedRollBackTaskList(svrName string) (list []tables.TableTaskInfo, err error) {
	err = d.db.Where("smt_status=? AND svr_name=?", tables.SmtStatusNeedToRollback, svrName).
		Order("parent_account_id,id DESC").Find(&list).Error
	return
}

func (d *DbDao) UpdateSmtRecordToRollbackComplete(taskId string, list []tables.TableSmtRecordInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.TableTaskInfo{}).
			Where("task_id=? AND smt_status=?", taskId, tables.SmtStatusNeedToRollback).
			Updates(map[string]interface{}{
				"smt_status": tables.SmtStatusRollbackComplete,
			}).Error; err != nil {
			return err
		}
		for i, _ := range list {
			if err := tx.Where("account_id=? AND nonce=? AND record_type=? AND task_id!=?",
				list[i].AccountId, list[i].Nonce, tables.RecordTypeClosed, taskId).
				Delete(&tables.TableSmtRecordInfo{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(tables.TableSmtRecordInfo{}).
			Where("task_id=?", taskId).
			Updates(map[string]interface{}{
				"record_type": tables.RecordTypeClosed,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) UpdateTaskCompleteWithDiffCustomScriptHash(taskId string, list []tables.TableSmtRecordInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.TableTaskInfo{}).
			Where("task_id=? AND smt_status=?", taskId, tables.SmtStatusNeedToWrite).
			Updates(map[string]interface{}{
				"smt_status": tables.SmtStatusRollbackComplete,
			}).Error; err != nil {
			return err
		}
		for i, _ := range list {
			if err := tx.Where("account_id=? AND nonce=? AND record_type=? AND task_id!=?",
				list[i].AccountId, list[i].Nonce, tables.RecordTypeClosed, taskId).
				Delete(&tables.TableSmtRecordInfo{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(tables.TableSmtRecordInfo{}).
			Where("task_id=?", taskId).
			Updates(map[string]interface{}{
				"record_type": tables.RecordTypeClosed,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) UpdateSmtRecordToNeedToWrite(taskId string, retry int) error {
	return d.db.Model(tables.TableTaskInfo{}).
		Where("task_id=? AND smt_status=?", taskId, tables.SmtStatusNeedToRollback).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusNeedToWrite,
			"tx_status":  tables.TxStatusUnSend,
			"retry":      retry,
		}).Error
}

func (d *DbDao) GetNeedToConfirmOtherTx(svrName string) (list []tables.TableTaskInfo, err error) {
	err = d.db.Where("smt_status=? AND tx_status=? AND svr_name=?",
		tables.SmtStatusNeedToWrite, tables.TxStatusCommitted, svrName).
		Order("block_number").Find(&list).Error
	return
}

func (d *DbDao) UpdateSmtStatusToWriteComplete(taskId string) error {
	return d.db.Model(tables.TableTaskInfo{}).
		Where("task_id=? AND smt_status=? AND tx_status=?",
			taskId, tables.SmtStatusNeedToWrite, tables.TxStatusCommitted).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusWriteComplete,
		}).Error
}

func (d *DbDao) UpdateSmtStatus(taskId string, smtStatus tables.SmtStatus) error {
	return d.db.Model(tables.TableTaskInfo{}).Where("task_id=?", taskId).
		Updates(map[string]interface{}{
			"smt_status": smtStatus,
		}).Error
}

func (d *DbDao) UpdateSmtRecordOutpoint(taskId, refOutpoint, outpoint string) error {
	return d.db.Model(tables.TableTaskInfo{}).Where("task_id=?", taskId).
		Updates(map[string]interface{}{
			"ref_outpoint": refOutpoint,
			"outpoint":     outpoint,
		}).Error
}

func (d *DbDao) GetNeedToDoTaskListByAction(svrName string, action common.DasAction) (list []tables.TableTaskInfo, err error) {
	smtStatus := []tables.SmtStatus{tables.SmtStatusNeedToWrite, tables.SmtStatusWriting}
	err = d.db.Where("action=? AND task_type=? AND smt_status IN(?) AND tx_status=? AND svr_name=?",
		action, tables.TaskTypeDelegate, smtStatus, tables.TxStatusUnSend, svrName).Limit(100).
		Order("parent_account_id,id").
		Find(&list).Error
	return
}

func (d *DbDao) GetTaskByRefOutpointAndOutpoint(refOutpoint, outpoint string) (task tables.TableTaskInfo, err error) {
	err = d.db.Where("ref_outpoint=? AND outpoint=? AND smt_status=? AND tx_status=?",
		refOutpoint, outpoint, tables.SmtStatusWriteComplete, tables.TxStatusPending).
		Order("id DESC").Find(&task).Error
	return
}

func (d *DbDao) CreateChainTask(task *tables.TableTaskInfo, list []tables.TableSmtRecordInfo, selfTaskId string) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if selfTaskId != "" {
			if err := tx.Where("task_id=?", selfTaskId).
				Delete(tables.TableTaskInfo{}).Error; err != nil {
				return err
			}
			if err := tx.Where("task_id=?", selfTaskId).
				Delete(tables.TableSmtRecordInfo{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&task).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&list).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) UpdateToChainTask(taskId string, blockNumber uint64) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.TableTaskInfo{}).Where("task_id=?", taskId).
			Updates(map[string]interface{}{
				"task_type":    tables.TaskTypeChain,
				"smt_status":   tables.SmtStatusNeedToWrite,
				"tx_status":    tables.TxStatusCommitted,
				"block_number": blockNumber,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(tables.TableSmtRecordInfo{}).
			Where("task_id=? AND record_type=?", taskId, tables.RecordTypeDefault).
			Updates(map[string]interface{}{
				"record_type": tables.RecordTypeChain,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateTaskByDasActionEnableSubAccount(task *tables.TableTaskInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ref_outpoint='' AND outpoint=?", task.Outpoint).
			Delete(&tables.TableTaskInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateTaskByConfigSubAccountCustomScript(task *tables.TableTaskInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ref_outpoint='' AND outpoint=?", task.Outpoint).
			Delete(&tables.TableTaskInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateTaskByProfitWithdraw(task *tables.TableTaskInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ref_outpoint='' AND outpoint=?", task.Outpoint).
			Delete(&tables.TableTaskInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) CreateTask(task *tables.TableTaskInfo) error {
	return d.db.Create(&task).Error
}

func (d *DbDao) GetTaskByTaskId(taskId string) (task tables.TableTaskInfo, err error) {
	err = d.db.Where("task_id=?", taskId).Find(&task).Error
	return
}

// (1,0)(2,1)(0,2)(3,?)
func (d *DbDao) GetTaskInProgress(parentAccountId string) (list []tables.TableTaskInfo, err error) {
	sql := fmt.Sprintf(`
SELECT * FROM %s WHERE parent_account_id=? AND smt_status=? AND tx_status=?
UNION ALL
SELECT * FROM %s WHERE parent_account_id=? AND smt_status=? AND tx_status=?
UNION ALL
SELECT * FROM %s WHERE parent_account_id=? AND smt_status=? AND tx_status=?
UNION ALL
SELECT * FROM %s WHERE parent_account_id=? AND smt_status=?
`, tables.TableNameTaskInfo, tables.TableNameTaskInfo, tables.TableNameTaskInfo, tables.TableNameTaskInfo)
	err = d.db.Raw(sql,
		parentAccountId, tables.SmtStatusWriting, tables.TxStatusUnSend,
		parentAccountId, tables.SmtStatusWriteComplete, tables.TxStatusPending,
		parentAccountId, tables.SmtStatusNeedToWrite, tables.TxStatusCommitted,
		parentAccountId, tables.SmtStatusNeedToRollback).Find(&list).Error
	return
}

func (d *DbDao) GetTaskByOutpointWithParentAccountId(parentAccountId, outpoint string) (task tables.TableTaskInfo, err error) {
	err = d.db.Where("parent_account_id=? AND outpoint=? AND task_type=? AND smt_status=?",
		parentAccountId, outpoint, tables.TaskTypeChain, tables.SmtStatusWriteComplete).Find(&task).Error
	return
}

func (d *DbDao) GetTaskByOutpointWithParentAccountIdForConfirm(parentAccountId, outpoint string) (task tables.TableTaskInfo, err error) {
	err = d.db.Where("parent_account_id=? AND outpoint=? AND task_type=? AND smt_status=? AND tx_status=?",
		parentAccountId, outpoint, tables.TaskTypeChain, tables.SmtStatusNeedToWrite, tables.TxStatusCommitted).Find(&task).Error
	return
}

func (d *DbDao) UpdateTaskStatusToRollback(ids []uint64) error {
	return d.db.Model(tables.TableTaskInfo{}).
		Where("id IN(?) AND smt_status=? AND tx_status=?", ids, tables.SmtStatusWriting, tables.TxStatusUnSend).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusNeedToRollback,
		}).Error
}

func (d *DbDao) UpdateTaskStatusToRollbackWithBalanceErr(taskId string) error {
	return d.db.Model(tables.TableTaskInfo{}).
		Where("task_id=? AND smt_status=? AND tx_status=?", taskId, tables.SmtStatusNeedToWrite, tables.TxStatusUnSend).
		Updates(map[string]interface{}{
			"smt_status": tables.SmtStatusNeedToRollback,
		}).Error
}

func (d *DbDao) GetTaskInfoByParentAccountIdWithAction(parentAccountId string, action common.DasAction) (task tables.TableTaskInfo, err error) {
	taskType := []tables.TaskType{tables.TaskTypeNormal, tables.TaskTypeChain}
	err = d.db.Where("parent_account_id=? AND action=? AND task_type IN(?)",
		parentAccountId, action, taskType).
		Order("id DESC").Limit(1).Find(&task).Error
	return
}

func (d *DbDao) GetLatestTaskByParentAccountId(parentAccountId string, limit int) (list []tables.TableTaskInfo, err error) {
	err = d.db.Where("parent_account_id=? AND task_type=? AND action!=?",
		parentAccountId, tables.TaskTypeChain, common.DasActionEnableSubAccount).
		Order("id DESC").Limit(limit).Find(&list).Error
	return
}

func (d *DbDao) UpdateTxStatusByOutpoint(outpoint string, txStatus tables.TxStatus) (err error) {
	err = d.db.Model(&tables.TableTaskInfo{}).Where("outpoint=?", outpoint).Update("tx_status", txStatus).Error
	return
}
