package block_parser

import (
	"das_sub_account/lb"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
)

func (b *BlockParser) DasActionEditSubAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version edit sub account tx")
		return
	}
	log.Info("DasActionEditSubAccount:", req.BlockNumber, req.TxHash)

	// get parentAccountId, refOutpoint, outpoint
	parentAccountId := common.Bytes2Hex(req.Tx.Outputs[0].Type.Args)
	refOutpoint := common.OutPointStruct2String(req.Tx.Inputs[0].PreviousOutput)
	outpoint := common.OutPoint2String(req.TxHash, 0)

	// get sub account
	taskInfo, smtRecordList, err := getTaskAndSmtRecords(b.Slb, &req, parentAccountId, refOutpoint, outpoint)
	if err != nil {
		resp.Err = fmt.Errorf("getTaskAndSmtRecords err: %s", err.Error())
		return
	}

	// get self task
	selfTask, err := b.DbDao.GetTaskByRefOutpointAndOutpoint(refOutpoint, outpoint)
	if err != nil {
		resp.Err = fmt.Errorf("GetTaskByRefOutpointAndOutpoint err: %s", err.Error())
		return
	}

	// add task and smt records
	if selfTask.TaskId != "" {
		// maybe rollback
		if err := b.DbDao.UpdateToChainTask(selfTask.TaskId, req.BlockNumber, 0); err != nil {
			resp.Err = fmt.Errorf("UpdateToChainTask err: %s", err.Error())
			return
		}
	} else {
		if err := b.DbDao.CreateChainTask(taskInfo, smtRecordList, selfTask.TaskId); err != nil {
			resp.Err = fmt.Errorf("CreateChainTask err: %s", err.Error())
			return
		}
	}

	return
}

func getTaskAndSmtRecords(slb *lb.LoadBalancing, req *FuncTransactionHandleReq, parentAccountId, refOutpoint, outpoint string) (*tables.TableTaskInfo, []tables.TableSmtRecordInfo, error) {
	svrName := ""
	if slb != nil {
		s := slb.GetServer(parentAccountId)
		svrName = s.Name
	}
	// get sub account
	var sanb witness.SubAccountNewBuilder
	subAccountMap, err := sanb.SubAccountNewMapFromTx(req.Tx) //witness.SubAccountBuilderMapFromTx(req.Tx)
	if err != nil {
		return nil, nil, fmt.Errorf("SubAccountNewMapFromTx err: %s", err.Error())
	}
	taskInfo := tables.TableTaskInfo{
		Id:              0,
		TaskId:          "",
		TaskType:        tables.TaskTypeChain,
		ParentAccountId: parentAccountId,
		Action:          req.Action,
		RefOutpoint:     refOutpoint,
		BlockNumber:     req.BlockNumber,
		Outpoint:        outpoint,
		Timestamp:       req.BlockTimestamp,
		SmtStatus:       tables.SmtStatusNeedToWrite,
		TxStatus:        tables.TxStatusCommitted,
		SvrName:         svrName,
	}
	taskInfo.InitTaskId()

	var smtRecordList []tables.TableSmtRecordInfo
	for _, v := range subAccountMap {
		record := tables.TableSmtRecordInfo{
			Id:              0,
			AccountId:       v.SubAccountData.AccountId,
			Nonce:           v.CurrentSubAccountData.Nonce,
			RecordType:      tables.RecordTypeChain,
			RecordBN:        req.BlockNumber,
			TaskId:          taskInfo.TaskId,
			Action:          req.Action,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			RegisterYears:   0,
			RegisterArgs:    "",
			EditKey:         string(v.EditKey),
			Signature:       common.Bytes2Hex(v.Signature),
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       req.BlockTimestamp,
		}
		switch req.Action {
		case common.DasActionCreateSubAccount:
			record.RegisterArgs = common.Bytes2Hex(v.SubAccountData.Lock.Args)
			record.RegisterYears = (v.SubAccountData.ExpiredAt - v.SubAccountData.RegisteredAt) / 31536000
		case common.DasActionEditSubAccount:
			if len(v.EditLockArgs) > 0 {
				record.EditArgs = common.Bytes2Hex(v.EditLockArgs)
			}
			if len(v.EditRecords) > 0 {
				recordsBys, err := json.Marshal(v.EditRecords)
				if err != nil {
					return nil, nil, fmt.Errorf("records json.Marshal err: %s", err.Error())
				}
				record.EditRecords = string(recordsBys)
			}
		}
		smtRecordList = append(smtRecordList, record)
	}
	return &taskInfo, smtRecordList, nil
}
