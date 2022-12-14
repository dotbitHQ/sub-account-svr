package task

import (
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"sync"
)

func (t *SmtTask) doBatchUpdateSubAccountTask(action common.DasAction) error {
	list, err := t.DbDao.GetNeedToDoTaskListByAction(config.Cfg.Slb.SvrName, action)
	if err != nil {
		return fmt.Errorf("GetNeedToDoTaskListByAction err: %s [%s]", err.Error(), action)
	}

	// group task list by ParentAccountId
	mapTaskList, mapTaskIdList := t.groupByParentAccountIdNew(list)

	// batch do update_sub_account tx
	var chanParentAccountId = make(chan string, 5)
	var wgTask sync.WaitGroup
	go func() {
		wgTask.Add(1)
		defer wgTask.Done()

		for parentAccountId, _ := range mapTaskList {
			chanParentAccountId <- parentAccountId
		}
		close(chanParentAccountId)
	}()

	for i := 0; i < 5; i++ {
		wgTask.Add(1)
		go func() {
			for {
				select {
				case parentAccountId, ok := <-chanParentAccountId:
					if !ok {
						wgTask.Done()
						return
					}
					if err := t.doUpdateSubAccountTaskDetail(action, parentAccountId, mapTaskList, mapTaskIdList); err != nil {
						log.Error("doUpdateSubAccountTaskDetail err: %s", err.Error())
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doUpdateSubAccountTaskDetail", err.Error()+parentAccountId)
					}
				}
			}
		}()
	}
	wgTask.Wait()
	return nil
}

func (t *SmtTask) doUpdateSubAccountTaskDetail(action common.DasAction, parentAccountId string, mapTaskList map[string][]tables.TableTaskInfo, mapTaskIdList map[string][]string) error {
	taskList, ok := mapTaskList[parentAccountId]
	if !ok {
		return fmt.Errorf("mapTaskList[%s] is nil", parentAccountId)
	}
	taskIdList, ok := mapTaskIdList[parentAccountId]
	if !ok {
		return fmt.Errorf("mapTaskIdList[%s] is nil", parentAccountId)
	}

	// check need roll back
	var needRollbackIds []uint64
	for _, v := range taskList {
		if v.SmtStatus != tables.SmtStatusNeedToWrite {
			needRollbackIds = append(needRollbackIds, v.Id)
		}
	}
	if len(needRollbackIds) > 0 {
		if err := t.DbDao.UpdateTaskStatusToRollback(needRollbackIds); err != nil {
			return fmt.Errorf("UpdateTaskStatusToRollback err: %s", err.Error())
		}
		return nil
	}

	if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
		log.Warn("SuspendMap:", parentAccountId)
		return nil
	}

	// get smt records
	taskMap, subAccountIds, err := t.getTaskMap(taskIdList)
	if err != nil {
		return fmt.Errorf("getTaskMap err: %s", err.Error())
	}

	// check nonce
	hasDiffNonce, err := t.doCheckNonceNew(taskMap, subAccountIds)
	if err != nil {
		return fmt.Errorf("doCheckNonce err: %s", err.Error())
	} else if hasDiffNonce {
		log.Warn("doCheckNonce:", parentAccountId)
		return nil
	}

	// do check
	resCheck, err := t.TxTool.DoCheckBeforeBuildTx(parentAccountId)
	if err != nil {
		if resCheck != nil && resCheck.Continue {
			log.Info("CheckInProgressTask: task in progress", parentAccountId)
			return nil
		}
		return fmt.Errorf("DoCheckBeforeBuildTx err: %s", err.Error())
	}

	// do check custom script
	if tId, customScriptOk := t.TxTool.DoCheckCustomScriptHashNew(resCheck.SubAccountLiveCell, taskList); !customScriptOk {
		log.Error("DoCheckCustomScriptHash err:", tId)
		if err := t.DbDao.UpdateTaskCompleteWithDiffCustomScriptHash(tId, taskMap[tId]); err != nil {
			return fmt.Errorf("UpdateTaskCompleteWithDiffCustomScriptHash err: %s", err.Error())
		}
		return fmt.Errorf("DoCheckCustomScriptHash err: %s", tId)
	}

	// get account
	parentAccount, err := t.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	}

	// do task detail
	if err := t.doTaskDetail(&paramDoTaskDetail{
		action:             action,
		taskList:           taskList,
		taskMap:            taskMap,
		account:            &parentAccount,
		subAccountLiveCell: resCheck.SubAccountLiveCell,
		baseInfo:           resCheck.BaseInfo,
		subAccountIds:      subAccountIds,
	}); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			log.Info("doTaskDetail: task in progress", parentAccountId)
			return nil
		}
		return fmt.Errorf("doTaskDetail err: %s", err.Error())
	}

	return nil
}

func (t *SmtTask) doUpdateSubAccountTask(action common.DasAction) error {
	list, err := t.DbDao.GetNeedToDoTaskListByAction(config.Cfg.Slb.SvrName, action)
	if err != nil {
		return fmt.Errorf("GetNeedToDoTaskListByAction err: %s [%s]", err.Error(), action)
	}

	// group task list by ParentAccountId
	mapTaskList, mapTaskIdList := t.groupByParentAccountIdNew(list)
	// do task
	for parentAccountId, taskList := range mapTaskList {
		taskIdList, _ := mapTaskIdList[parentAccountId]

		// check need roll back
		var needRollbackIds []uint64
		for _, v := range taskList {
			if v.SmtStatus != tables.SmtStatusNeedToWrite {
				needRollbackIds = append(needRollbackIds, v.Id)
			}
		}
		if len(needRollbackIds) > 0 {
			if err := t.DbDao.UpdateTaskStatusToRollback(needRollbackIds); err != nil {
				return fmt.Errorf("UpdateTaskStatusToRollback err: %s", err.Error())
			}
			continue
		}

		if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
			log.Warn("SuspendMap:", parentAccountId)
			continue
		}

		// get smt records
		taskMap, subAccountIds, err := t.getTaskMap(taskIdList)
		if err != nil {
			return fmt.Errorf("getTaskMap err: %s", err.Error())
		}

		// check nonce
		hasDiffNonce, err := t.doCheckNonceNew(taskMap, subAccountIds)
		if err != nil {
			return fmt.Errorf("doCheckNonce err: %s", err.Error())
		} else if hasDiffNonce {
			log.Warn("doCheckNonce:", parentAccountId)
			continue
		}

		// do check
		resCheck, err := t.TxTool.DoCheckBeforeBuildTx(parentAccountId)
		if err != nil {
			if resCheck != nil && resCheck.Continue {
				log.Info("CheckInProgressTask: task in progress", parentAccountId)
				continue
			}
			return fmt.Errorf("DoCheckBeforeBuildTx err: %s", err.Error())
		}

		// do check custom script
		if tId, customScriptOk := t.TxTool.DoCheckCustomScriptHashNew(resCheck.SubAccountLiveCell, taskList); !customScriptOk {
			log.Error("DoCheckCustomScriptHash err:", tId)
			if err := t.DbDao.UpdateTaskCompleteWithDiffCustomScriptHash(tId, taskMap[tId]); err != nil {
				return fmt.Errorf("UpdateTaskCompleteWithDiffCustomScriptHash err: %s", err.Error())
			}
			return fmt.Errorf("DoCheckCustomScriptHash err: %s", tId)
		}

		// get account
		parentAccount, err := t.DbDao.GetAccountInfoByAccountId(parentAccountId)
		if err != nil {
			return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
		}

		// do task detail
		if err := t.doTaskDetail(&paramDoTaskDetail{
			action:             action,
			taskList:           taskList,
			taskMap:            taskMap,
			account:            &parentAccount,
			subAccountLiveCell: resCheck.SubAccountLiveCell,
			baseInfo:           resCheck.BaseInfo,
			subAccountIds:      subAccountIds,
		}); err != nil {
			if err == cache.ErrDistributedLockPreemption {
				log.Info("doTaskDetail: task in progress", parentAccountId)
				continue
			}
			return fmt.Errorf("doTaskDetail err: %s", err.Error())
		}
	}
	return nil
}

func (t *SmtTask) groupByParentAccountIdNew(list []tables.TableTaskInfo) (map[string][]tables.TableTaskInfo, map[string][]string) {
	var mapTaskList = make(map[string][]tables.TableTaskInfo)
	var mapTaskIdList = make(map[string][]string)

	maxTaskCount := 1

	for i, v := range list {
		if len(mapTaskList[v.ParentAccountId]) >= maxTaskCount {
			continue
		}
		mapTaskList[v.ParentAccountId] = append(mapTaskList[v.ParentAccountId], list[i])
		mapTaskIdList[v.ParentAccountId] = append(mapTaskIdList[v.ParentAccountId], list[i].TaskId)
	}
	return mapTaskList, mapTaskIdList
}

func (t *SmtTask) doCheckNonceNew(taskMap map[string][]tables.TableSmtRecordInfo, subAccountIds []string) (bool, error) {
	log.Info("doCheckNonce:")
	subAccList, err := t.DbDao.GetAccountListByAccountIds(subAccountIds)
	if err != nil {
		return false, fmt.Errorf("GetAccountListByAccountIds err: %s", err.Error())
	}
	var nonceMap = make(map[string]uint64)
	for _, v := range subAccList {
		nonceMap[v.AccountId] = v.Nonce
	}

	hasDiffNonce := false
	for taskId, records := range taskMap {
		var diffNonceList []tables.TableSmtRecordInfo
		for i, v := range records {
			nonce, ok := nonceMap[v.AccountId]
			if v.SubAction == common.SubActionCreate && ok {
				diffNonceList = append(diffNonceList, records[i])
				log.Info("doCheckNonce create diff:", v.Id, v.AccountId, v.Nonce)
			} else if ok && nonce >= v.Nonce {
				log.Info("doCheckNonce other diff:", v.Id, v.AccountId, v.Nonce)
				diffNonceList = append(diffNonceList, records[i])
			}
		}
		if len(diffNonceList) > 0 {
			hasDiffNonce = true
		}
		if err := t.DbDao.UpdateRecordsToClosed(taskId, diffNonceList, len(diffNonceList) == len(records)); err != nil {
			return false, fmt.Errorf("UpdateRecordsToClosed err: %s", err.Error())
		}
	}
	return hasDiffNonce, nil
}
