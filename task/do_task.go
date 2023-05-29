package task

import (
	"bytes"
	"context"
	"das_sub_account/cache"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
)

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
	tree := smt.NewSmtSrv(t.SmtServerUrl, parentAccountId)
	// check root
	currentRoot, err := tree.GetSmtRoot()
	if err != nil {
		log.Warn("getSmtRoot error: ", err)
		return fmt.Errorf("GetOldSubAccount err: %s", err.Error())
	}
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.subAccountLiveCell.OutputData)
	log.Warn("Compare root:", parentAccountId, common.Bytes2Hex(currentRoot), common.Bytes2Hex(subDataDetail.SmtRoot))
	if bytes.Compare(currentRoot, subDataDetail.SmtRoot) != 0 {
		log.Warn("currentRoot:", currentRoot, "chain_root: ", parentAccountId)
		return fmt.Errorf("smt root diff: %s", parentAccountId)
	}

	// build tx
	res, err := t.TxTool.BuildTxsForUpdateSubAccount(&txtool.ParamBuildTxs{
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
		return fmt.Errorf("BuildTxsForUpdateSubAccount err: %s", err.Error())
	}

	// send txs
	for i, _ := range p.taskList {
		txBys, _ := res.DasTxBuilderList[i].Transaction.Serialize()
		log.Info("doTaskDetail:", len(txBys))
		if hash, err := res.DasTxBuilderList[i].SendTransaction(); err != nil {
			return fmt.Errorf("SendTransaction err: %s", err.Error())
		} else {
			log.Info("SendTransaction:", hash.String(), p.taskList[i].TaskId)
			if err := t.DbDao.UpdateTaskTxStatusToPending(p.taskList[i].TaskId); err != nil {
				log.Error("UpdateTaskTxStatusToPending err: %s", err.Error())
			}
		}
		//time.Sleep(time.Second)
	}
	return nil
}

func DoSign(action common.DasAction, signList []txbuilder.SignData, privateKey string, compress bool) error {
	for i, signData := range signList {
		var signRes []byte
		var err error

		if signData.SignMsg == "" {
			signList[i].SignMsg = ""
			continue
		}

		signMsg := []byte(signData.SignMsg)

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
		case common.DasAlgorithmIdDogeChain:
			signRes, err = sign.DogeSignature(signMsg, privateKey, compress)
			if err != nil {
				return fmt.Errorf("sign.DogeSignature err: %s", err.Error())
			}
		default:
			return fmt.Errorf("not supported sign type[%d]", signData.SignType)
		}
		signList[i].SignMsg = common.Bytes2Hex(signRes)
	}
	return nil
}
