package task

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"time"
)

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
	maxUpdateCount := 100
	if config.Cfg.Das.MaxUpdateCount > 0 {
		maxUpdateCount = config.Cfg.Das.MaxUpdateCount
	}
	timestamp := time.Now().Add(-time.Minute).UnixNano() / 1e6
	for k, v := range mapSmtRecordList {
		if timestamp < v[0].Timestamp && len(v) < maxUpdateCount {
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
		ids = make([]uint64, 0)
		for _, smtRecord := range smtRecordList {
			if count == maxUpdateCount {
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
			//log.Info("doUpdateDistribution:", smtRecord.Id, count, lastMintSignId)
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
