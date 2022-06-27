package task

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

func (t *SmtTask) doDistribution() error {
	if err := t.doEditDistribution(); err != nil {
		return fmt.Errorf("doEditDistribution err: %s", err.Error())
	}
	return nil
}

func (t *SmtTask) doEditDistribution() error {
	action := common.DasActionEditSubAccount
	list, err := t.DbDao.GetNeedDoDistributionRecordList(action)
	if err != nil {
		return fmt.Errorf("GetNeedDoDistributionRecordList err: %s", err.Error())
	}
	if len(list) == 0 {
		return nil
	}
	var idsList [][]uint64
	var ids []uint64
	// distribution
	var taskList []tables.TableTaskInfo
	lastParentAccountId, count := "", 0
	for i, v := range list {
		addTask := false
		if v.ParentAccountId != lastParentAccountId {
			addTask = true
		} else if count >= config.Cfg.Das.MaxEditCount {
			addTask = true
		} else {
			count++
		}
		if addTask {
			lastParentAccountId = v.ParentAccountId
			count = 0
			tmp := tables.TableTaskInfo{
				Id:              0,
				TaskId:          "",
				TaskType:        tables.TaskTypeDelegate,
				ParentAccountId: lastParentAccountId,
				Action:          action,
				RefOutpoint:     "",
				BlockNumber:     0,
				Outpoint:        "",
				Timestamp:       time.Now().UnixNano() / 1e6,
				SmtStatus:       tables.SmtStatusNeedToWrite,
				TxStatus:        tables.TxStatusUnSend,
			}
			tmp.InitTaskId()
			taskList = append(taskList, tmp)
			if len(ids) > 0 {
				idsList = append(idsList, ids)
			}
			ids = make([]uint64, 0)
		}
		ids = append(ids, list[i].Id)
	}
	if len(ids) > 0 {
		idsList = append(idsList, ids)
	}
	if err := t.DbDao.UpdateTaskDistribution(taskList, idsList); err != nil {
		return fmt.Errorf("UpdateTaskDistribution err: %s", err.Error())
	}
	return nil
}

func (t *SmtTask) doMintDistribution() error {
	if err := t.doCreateDistribution(); err != nil {
		return fmt.Errorf("doCreateDistribution err: %s", err.Error())
	}
	return nil
}

func (t *SmtTask) doCreateDistribution() error {
	action := common.DasActionCreateSubAccount
	list, err := t.DbDao.GetNeedDoDistributionRecordList(action)
	if err != nil {
		return fmt.Errorf("GetNeedDoDistributionRecordList err: %s", err.Error())
	}
	if len(list) == 0 {
		return nil
	}
	var idsList [][]uint64
	var ids []uint64
	// distribution
	var taskList []tables.TableTaskInfo
	lastParentAccountId, count := "", 0
	for i, v := range list {
		addTask := false
		if v.ParentAccountId != lastParentAccountId {
			addTask = true
		} else if count >= config.Cfg.Das.MaxCreateCount {
			addTask = true
		} else {
			count++
		}
		if addTask {
			lastParentAccountId = v.ParentAccountId
			count = 0
			tmp := tables.TableTaskInfo{
				Id:              0,
				TaskId:          "",
				TaskType:        tables.TaskTypeDelegate,
				ParentAccountId: lastParentAccountId,
				Action:          action,
				RefOutpoint:     "",
				BlockNumber:     0,
				Outpoint:        "",
				Timestamp:       time.Now().UnixNano() / 1e6,
				SmtStatus:       tables.SmtStatusNeedToWrite,
				TxStatus:        tables.TxStatusUnSend,
			}
			tmp.InitTaskId()
			taskList = append(taskList, tmp)
			if len(ids) > 0 {
				idsList = append(idsList, ids)
			}
			ids = make([]uint64, 0)
		}
		ids = append(ids, list[i].Id)
	}
	if len(ids) > 0 {
		idsList = append(idsList, ids)
	}
	if err := t.DbDao.UpdateTaskDistribution(taskList, idsList); err != nil {
		return fmt.Errorf("UpdateTaskDistribution err: %s", err.Error())
	}
	return nil
}
