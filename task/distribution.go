package task

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
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
	list, err := t.DbDao.GetNeedDoDistributionRecordList(config.Cfg.Slb.SvrName, action)
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
				SvrName:         v.SvrName,
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
	list, err := t.DbDao.GetNeedDoDistributionRecordList(config.Cfg.Slb.SvrName, action)
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

			// check custom-script
			subAccLiveCell, err := t.DasCore.GetSubAccountCell(lastParentAccountId)
			if err != nil {
				return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
			}
			subAccDetail := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
			customScripHash := ""
			if subAccDetail.HasCustomScriptArgs() {
				customScripHash = subAccDetail.ArgsAndConfigHash()
			}

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
				CustomScripHash: customScripHash,
				SvrName:         v.SvrName,
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

// update-sub-account
func (t *SmtTask) doUpdateDistribution() error {
	action := common.DasActionUpdateSubAccount
	list, err := t.DbDao.GetNeedDoDistributionRecordListNew(config.Cfg.Slb.SvrName, action)
	if err != nil {
		return fmt.Errorf("GetNeedDoDistributionRecordList err: %s", err.Error())
	}
	if len(list) == 0 {
		return nil
	}
	var mapSmtRecordList = make(map[string][]tables.TableSmtRecordInfo)
	for i, v := range list {
		mapSmtRecordList[v.ParentAccountId] = append(mapSmtRecordList[v.ParentAccountId], list[i])
	}
	// distribution time
	timestamp := time.Now().Add(-time.Minute).UnixNano() / 1e6
	for k, v := range mapSmtRecordList {
		if timestamp < v[0].Timestamp && len(v) < config.Cfg.Das.MaxCreateCount {
			delete(mapSmtRecordList, k)
		}
	}
	if len(mapSmtRecordList) == 0 {
		return nil
	}
	// distribution
	var taskList []tables.TableTaskInfo
	var idsList [][]uint64
	var ids []uint64
	log.Info("doUpdateDistribution:", len(mapSmtRecordList))
	for _, smtRecordList := range mapSmtRecordList {
		// check custom-script
		subAccLiveCell, err := t.DasCore.GetSubAccountCell(smtRecordList[0].ParentAccountId)
		if err != nil {
			return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
		}
		subAccDetail := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
		customScripHash := ""
		if subAccDetail.HasCustomScriptArgs() {
			customScripHash = subAccDetail.ArgsAndConfigHash()
		}
		//
		count := 0
		lastMintSignId := ""
		addTask := true
		for _, smtRecord := range smtRecordList {
			if count == config.Cfg.Das.MaxCreateCount {
				addTask = true
				count = 0
			} else if lastMintSignId != "" && smtRecord.MintSignId != "" && lastMintSignId != smtRecord.MintSignId {
				addTask = true
				count = 0
			}
			if smtRecord.MintSignId != "" {
				lastMintSignId = smtRecord.MintSignId
			}
			count++

			if addTask {
				taskInfo := tables.TableTaskInfo{
					Id:              0,
					SvrName:         smtRecord.SvrName,
					TaskId:          "",
					TaskType:        tables.TaskTypeDelegate,
					ParentAccountId: smtRecord.ParentAccountId,
					Action:          action,
					RefOutpoint:     "",
					BlockNumber:     0,
					Outpoint:        "",
					Timestamp:       time.Now().UnixNano() / 1e6,
					SmtStatus:       tables.SmtStatusNeedToWrite,
					TxStatus:        tables.TxStatusUnSend,
					Retry:           0,
					CustomScripHash: customScripHash,
				}
				taskInfo.InitTaskId()
				taskList = append(taskList, taskInfo)

				if len(ids) > 0 {
					idsList = append(idsList, ids)
				}
				ids = make([]uint64, 0)
				addTask = false
			}
			ids = append(ids, smtRecord.Id)
		}
		if len(ids) > 0 {
			idsList = append(idsList, ids)
		}
	}

	if err := t.DbDao.UpdateTaskDistribution(taskList, idsList); err != nil {
		return fmt.Errorf("UpdateTaskDistribution err: %s", err.Error())
	}
	return nil
}
