package task

import (
	"bytes"
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"time"
)

func (t *SmtTask) doTask(action common.DasAction) error {
	list, err := t.DbDao.GetNeedToDoTaskListByAction(action)
	if err != nil {
		return fmt.Errorf("GetNeedToDoTaskListByAction err: %s [%s]", err.Error(), action)
	}

	// group task list by ParentAccountId
	mapTaskList, mapTaskIdList := t.groupByParentAccountId(action, list)

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
		hasDiffNonce, err := t.doCheckNonce(taskMap, subAccountIds)
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

func (t *SmtTask) doCheckNonce(taskMap map[string][]tables.TableSmtRecordInfo, subAccountIds []string) (bool, error) {
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
			if v.Action == common.DasActionCreateSubAccount && ok {
				diffNonceList = append(diffNonceList, records[i])
				log.Info("doCheckNonce diff:", v.Id, v.AccountId, v.Nonce)
			} else if ok && nonce >= v.Nonce {
				log.Info("doCheckNonce diff:", v.Id, v.AccountId, v.Nonce)
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

func (t *SmtTask) groupByParentAccountId(action common.DasAction, list []tables.TableTaskInfo) (map[string][]tables.TableTaskInfo, map[string][]string) {
	var mapTaskList = make(map[string][]tables.TableTaskInfo)
	var mapTaskIdList = make(map[string][]string)

	maxTaskCount := 1
	switch action {
	case common.DasActionEditSubAccount:
		maxTaskCount = config.Cfg.Das.MaxEditTaskCount
		if maxTaskCount > 5 {
			maxTaskCount = 5
		} else if maxTaskCount < 1 {
			maxTaskCount = 1
		}
	}

	for i, v := range list {
		if len(mapTaskList[v.ParentAccountId]) >= maxTaskCount {
			continue
		}
		mapTaskList[v.ParentAccountId] = append(mapTaskList[v.ParentAccountId], list[i])
		mapTaskIdList[v.ParentAccountId] = append(mapTaskIdList[v.ParentAccountId], list[i].TaskId)
	}
	return mapTaskList, mapTaskIdList
}

func (t *SmtTask) getTaskMap(taskIdList []string) (map[string][]tables.TableSmtRecordInfo, []string, error) {
	records, err := t.DbDao.GetSmtRecordListByTaskIds(taskIdList, tables.RecordTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSmtRecordListByTaskIds err: %s", err.Error())
	}
	var taskMap = make(map[string][]tables.TableSmtRecordInfo)
	var subAccountIds []string
	for i, record := range records {
		taskMap[record.TaskId] = append(taskMap[record.TaskId], records[i])
		subAccountIds = append(subAccountIds, record.AccountId)
	}
	return taskMap, subAccountIds, nil
}

type paramDoTaskDetail struct {
	action             common.DasAction
	taskList           []tables.TableTaskInfo
	taskMap            map[string][]tables.TableSmtRecordInfo
	subAccountIds      []string
	account            *tables.TableAccountInfo // parent account
	subAccountLiveCell *indexer.LiveCell
	baseInfo           *txtool.BaseInfo
}

func (t *SmtTask) doTaskDetail(p *paramDoTaskDetail) error {
	parentAccountId := p.taskList[0].ParentAccountId

	// lock smt and unlock
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

	// get history
	valueMap, subAccountBuilderMap, err := t.TxTool.GetOldSubAccount(p.subAccountIds, p.action)
	if err != nil {
		return fmt.Errorf("GetOldSubAccount err: %s", err.Error())
	}

	// get smt tree
	mongoStore := smt.NewMongoStore(t.Ctx, t.Mongo, config.Cfg.DB.Mongo.SmtDatabase, parentAccountId)
	tree := smt.NewSparseMerkleTree(mongoStore)

	// check root
	currentRoot, _ := tree.Root()
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.subAccountLiveCell.OutputData)
	log.Warn("Compare root:", parentAccountId, common.Bytes2Hex(currentRoot), common.Bytes2Hex(subDataDetail.SmtRoot))
	if bytes.Compare(currentRoot, subDataDetail.SmtRoot) != 0 {
		return fmt.Errorf("smt root diff: %s", parentAccountId)
	}

	// build tx
	res, err := t.TxTool.BuildTxs(&txtool.ParamBuildTxs{
		TaskList:             p.taskList,
		TaskMap:              p.taskMap,
		Account:              p.account,
		SubAccountLiveCell:   p.subAccountLiveCell,
		Tree:                 tree,
		BaseInfo:             p.baseInfo,
		BalanceDasLock:       t.TxTool.ServerScript,
		BalanceDasType:       nil,
		SubAccountIds:        p.subAccountIds,
		SubAccountValueMap:   valueMap,
		SubAccountBuilderMap: subAccountBuilderMap,
	})
	if err != nil {
		return fmt.Errorf("BuildTxs err: %s", err.Error())
	}

	// do sign
	if p.action == common.DasActionCreateSubAccount {
		for i, _ := range res.DasTxBuilderList {
			signList, err := res.DasTxBuilderList[i].GenerateDigestListFromTx([]int{})
			if err != nil {
				return fmt.Errorf("GenerateDigestListFromTx err: %s", err.Error())
			}
			log.Info("GenerateDigestListFromTx:", toolib.JsonString(signList))
			if err := DoSign("", signList, config.Cfg.Server.ManagerPrivateKey); err != nil {
				return fmt.Errorf("DoSign err: %s", err.Error())
			}
			if err := res.DasTxBuilderList[i].AddSignatureForTx(signList); err != nil {
				return fmt.Errorf("AddSignatureForTx err: %s", err.Error())
			}
		}
	}

	// send txs
	for i, _ := range p.taskList {
		if hash, err := res.DasTxBuilderList[i].SendTransaction(); err != nil {
			return fmt.Errorf("SendTransaction err: %s", err.Error())
		} else {
			log.Info("SendTransaction:", hash.String(), p.taskList[i].TaskId)
			if err := t.DbDao.UpdateTaskTxStatusToPending(p.taskList[i].TaskId); err != nil {
				log.Error("UpdateTaskTxStatusToPending err: %s", err.Error())
			}
		}
		time.Sleep(time.Second)
	}
	return nil
}

func DoSign(action common.DasAction, signList []txbuilder.SignData, privateKey string) error {
	for i, signData := range signList {
		var signRes []byte
		var err error

		if signData.SignMsg == "" {
			signList[i].SignMsg = ""
			continue
		}

		//var signMsg []byte
		//switch action {
		//case common.DasActionEditSubAccount:
		//	signMsg = []byte(signData.SignMsg)
		//default:
		//	signMsg = common.Hex2Bytes(signData.SignMsg)
		//}
		signMsg := common.Hex2Bytes(signData.SignMsg)

		switch signData.SignType {
		case common.DasAlgorithmIdCkb, common.DasAlgorithmIdEth, common.DasAlgorithmIdEth712:
			signRes, err = sign.PersonalSignature(signMsg, privateKey)
			if err != nil {
				return fmt.Errorf("sign.PersonalSignature err: %s", err.Error())
			}
		case common.DasAlgorithmIdTron:
			signRes, err = sign.TronSignature(true, signMsg, privateKey)
			if err != nil {
				return fmt.Errorf("sign.TronSignature err: %s", err.Error())
			}
		case common.DasAlgorithmIdEd25519:
			signRes = sign.Ed25519Signature(common.Hex2Bytes(privateKey), signMsg)
			signRes = append(signRes, []byte{1}...)
		default:
			return fmt.Errorf("not supported sign type[%d]", signData.SignType)
		}
		signList[i].SignMsg = common.Bytes2Hex(signRes)
	}
	return nil
}
