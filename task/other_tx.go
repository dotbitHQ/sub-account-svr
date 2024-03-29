package task

import (
	"bytes"
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/witness"
)

func (t *SmtTask) doConfirmOtherTx() error {
	list, err := t.DbDao.GetNeedToConfirmOtherTx(config.Cfg.Slb.SvrName)
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
					log.Warn("confirmOtherTx: errBlockNotSync:", err.Error())
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
	tree := smt.NewSmtSrv(t.SmtServerUrl, parentAccountId)
	// check root diff
	isUpdate := true
	contractSubAcc, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	subAccountLiveCell, err := t.TxTool.CheckSubAccountLiveCellForConfirm(contractSubAcc, parentAccountId)
	if err != nil {
		log.Warn("confirmOtherTx CheckSubAccountLiveCellForConfirm err:", err.Error())
	}

	if subAccountLiveCell != nil {
		currentRoot, err := tree.GetSmtRoot()
		if err != nil {
			return fmt.Errorf("tree.Root err: %s", err.Error())
		}
		subDataDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
		log.Warn("confirmOtherTx Compare root:", parentAccountId, common.Bytes2Hex(currentRoot), common.Bytes2Hex(subDataDetail.SmtRoot))
		if bytes.Compare(currentRoot, subDataDetail.SmtRoot) == 0 {
			isUpdate = false
		}
	}

	// update
	var smtKv []smt.SmtKv
	if isUpdate {

		for i, smtInfo := range smtInfoList {
			key := smt.AccountIdToSmtH256(smtInfo.AccountId)
			value := common.Hex2Bytes(smtInfo.LeafDataHash)

			log.Info("confirmOtherTx:", task.TaskId, len(smtInfoList), "-", i)
			//log.Info("confirmOtherTx key:", common.Bytes2Hex(key))
			//log.Info("confirmOtherTx value:", common.Bytes2Hex(value))
			smtKv = append(smtKv, smt.SmtKv{
				key,
				value,
			})

		}
	}
	if len(smtKv) > 0 {
		opt := smt.SmtOpt{GetProof: false, GetRoot: false}
		_, err := tree.UpdateSmt(smtKv, opt)
		if err != nil {
			return fmt.Errorf("tree.Update err: %s", err.Error())
		}

	}

	if root, err := tree.GetSmtRoot(); err != nil {
		return fmt.Errorf("tree.Root err: %s", err.Error())
	} else {
		log.Info("confirmOtherTx tree.Root():", task.ParentAccountId, common.Bytes2Hex(root))
	}

	// 0,2 -> 2,2
	if err := t.DbDao.UpdateSmtStatusToWriteComplete(task.TaskId); err != nil {
		return fmt.Errorf("UpdateSmtStatusToWriteComplete err: %s", err.Error())
	}
	return nil
}
