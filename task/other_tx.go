package task

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/smt"
)

func (t *SmtTask) doConfirmOtherTx() error {
	list, err := t.DbDao.GetNeedToConfirmOtherTx()
	if err != nil {
		return fmt.Errorf("GetNeedToConfirmOtherTx err: %s", err.Error())
	}

	// group by parent account id
	var mapTaskList = make(map[string][]tables.TableTaskInfo)
	for i, v := range list {
		mapTaskList[v.ParentAccountId] = append(mapTaskList[v.ParentAccountId], list[i])
	}

	// confirm
	for _, taskList := range mapTaskList {
		for i, _ := range taskList {
			if err := t.confirmOtherTx(&taskList[i]); err != nil {
				if err == cache.ErrDistributedLockPreemption {
					log.Info("confirmOtherTx: cache.ErrDistributedLockPreemption:", err.Error())
					break
				} else if err == errBlockNotSync {
					log.Info("confirmOtherTx: errBlockNotSync:", err.Error())
					break
				} else {
					return fmt.Errorf("confirmOtherTx err: %s", err.Error())
				}
			}
		}
	}

	return nil
}

var errBlockNotSync = errors.New("block not sync")

func (t *SmtTask) confirmOtherTx(task *tables.TableTaskInfo) error {
	parentAccountId := task.ParentAccountId

	// get records
	records, err := t.DbDao.GetChainSmtRecordListByTaskId(task.TaskId)
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

	// check
	if len(subAccountIds) != len(smtInfoList) {
		return errBlockNotSync
	}

	for _, v := range smtInfoList {
		if v.BlockNumber < task.BlockNumber {
			return errBlockNotSync
		}
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

	// tree
	mongoStore := smt.NewMongoStore(t.Ctx, t.Mongo, config.Cfg.DB.Mongo.SmtDatabase, parentAccountId)
	tree := smt.NewSparseMerkleTree(mongoStore)

	// update
	for _, smtInfo := range smtInfoList {
		key := smt.AccountIdToSmtH256(smtInfo.AccountId)
		value := common.Hex2Bytes(smtInfo.LeafDataHash)

		log.Info("confirmOtherTx:", smtInfo.AccountId)
		log.Info("confirmOtherTx key:", common.Bytes2Hex(key))
		log.Info("confirmOtherTx value:", common.Bytes2Hex(value))

		err = tree.Update(key, value)
		if err != nil {
			return fmt.Errorf("tree.Update err: %s", err.Error())
		}
	}

	if root, err := tree.Root(); err != nil {
		return fmt.Errorf("tree.Root err: %s", err.Error())
	} else {
		log.Info("confirmOtherTx CurrentRoot:", task.ParentAccountId, common.Bytes2Hex(root))
	}

	// 0,2 -> 2,2
	if err := t.DbDao.UpdateSmtStatusToWriteComplete(task.TaskId); err != nil {
		return fmt.Errorf("UpdateSmtStatusToWriteComplete err: %s", err.Error())
	}
	return nil
}
