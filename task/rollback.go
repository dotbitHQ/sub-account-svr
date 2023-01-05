package task

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/smt"
)

func (t *SmtTask) doRollback() error {
	list, err := t.DbDao.GetNeedRollBackTaskList(config.Cfg.Slb.SvrName)
	if err != nil {
		return fmt.Errorf("GetNeedRollBackTaskList err: %s", err.Error())
	}

	for i, _ := range list {
		if err := t.rollback(&list[i]); err != nil {
			if err == cache.ErrDistributedLockPreemption {
				log.Info("confirmOtherTx: cache.ErrDistributedLockPreemption:", err.Error())
				continue
			} else {
				return fmt.Errorf("confirmOtherTx err: %s", err.Error())
			}
		}
	}

	return nil
}

func (t *SmtTask) rollback(task *tables.TableTaskInfo) error {
	parentAccountId := task.ParentAccountId

	// get records
	records, err := t.DbDao.GetSmtRecordListByTaskId(task.TaskId)
	if err != nil {
		return fmt.Errorf("GetSmtRecordListByTaskId err: %s", err.Error())
	}
	var subAccountIds []string
	for _, v := range records {
		subAccountIds = append(subAccountIds, v.AccountId)
	}

	// get smt value
	smtInfoList, err := t.DbDao.GetSmtInfoBySubAccountIds(subAccountIds)
	if err != nil {
		return fmt.Errorf("GetSmtInfoBySubAccountIds err:%s", err.Error())
	}
	var subAccountValueMap = make(map[string]string)
	for _, v := range smtInfoList {
		subAccountValueMap[v.AccountId] = v.LeafDataHash
	}

	// lock smt and defer unlock
	if err := t.RC.LockWithRedis(parentAccountId); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			return err
		}
		return fmt.Errorf("LockWithRedis err: %s", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := t.RC.UnLockWithRedis(parentAccountId); err != nil {
			fmt.Println("UnLockWithRedis:", err.Error())
		}
		cancel()
	}()
	t.RC.DoLockExpire(ctx, parentAccountId)

	tree := smt.NewSmtSrv(t.SmtServerUrl, parentAccountId)
	// update
	var smtKv []smt.SmtKv
	for i, v := range subAccountIds {
		key := smt.AccountIdToSmtH256(v)
		value := smt.H256Zero()

		log.Info("rollback:", task.TaskId, len(subAccountIds), "-", i)
		//log.Info("rollback key:", common.Bytes2Hex(key))
		//log.Info("rollback value:", common.Bytes2Hex(value))

		if subAccountValue, ok := subAccountValueMap[v]; ok {
			value = common.Hex2Bytes(subAccountValue)
		}
		smtKv = append(smtKv, smt.SmtKv{
			key,
			value,
		})
	}
	opt := smt.SmtOpt{GetProof: false, GetRoot: false}
	_, err = tree.UpdateSmt(smtKv, opt)
	if err != nil {
		return fmt.Errorf("tree.Update err: %s", err.Error())
	}

	//if _, err = tree.Root(); err != nil {
	//	return fmt.Errorf("tree.Root err: %s", err.Error())
	//} else {
	//	//log.Info("rollback CurrentRoot:", task.ParentAccountId, common.Bytes2Hex(root))
	//}

	if task.TaskType == tables.TaskTypeDelegate && task.Retry < t.MaxRetry {
		if err := t.DbDao.UpdateSmtRecordToNeedToWrite(task.TaskId, task.Retry+1); err != nil {
			return fmt.Errorf("UpdateSmtRecordToNeedToWrite err: %s", err.Error())
		}
	} else {
		if err := t.DbDao.UpdateSmtRecordToRollbackComplete(task.TaskId, records); err != nil {
			return fmt.Errorf("UpdateSmtRecordToRollbackComplete err: %s", err.Error())
		}
	}

	return nil
}
