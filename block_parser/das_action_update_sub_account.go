package block_parser

import (
	"das_sub_account/config"
	"das_sub_account/lb"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
)

func (b *BlockParser) DasActionUpdateSubAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionUpdateSubAccount:", req.BlockNumber, req.TxHash)

	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
	}
	parentAccountId, outpoint, refOutpoint := "", "", ""
	for i, v := range req.Tx.Outputs {
		if v.Type != nil && contractSub.IsSameTypeId(v.Type.CodeHash) {
			parentAccountId = common.Bytes2Hex(v.Type.Args)
			outpoint = common.OutPoint2String(req.TxHash, uint(i))
		}
	}
	for _, v := range req.Tx.Inputs {
		res, err := b.DasCore.Client().GetTransaction(b.Ctx, v.PreviousOutput.TxHash)
		if err != nil {
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	// get task , smt-record
	taskInfo, smtRecordList, err := b.getTaskAndSmtRecordsNew(b.Slb, &req, parentAccountId, refOutpoint, outpoint)
	if err != nil {
		resp.Err = fmt.Errorf("getTaskAndSmtRecordsNew err: %s", err.Error())
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
		if err := b.DbDao.UpdateToChainTask(selfTask.TaskId, req.BlockNumber); err != nil {
			resp.Err = fmt.Errorf("UpdateToChainTask err: %s", err.Error())
			return
		}
	} else {
		if err := b.DbDao.CreateChainTask(taskInfo, smtRecordList, selfTask.TaskId); err != nil {
			resp.Err = fmt.Errorf("CreateChainTask err: %s", err.Error())
			return
		}
	}

	doNotifyDiscord(smtRecordList)
	b.doNotifyLark(smtRecordList)

	return
}

func (b *BlockParser) getTaskAndSmtRecordsNew(slb *lb.LoadBalancing, req *FuncTransactionHandleReq, parentAccountId, refOutpoint, outpoint string) (*tables.TableTaskInfo, []tables.TableSmtRecordInfo, error) {
	svrName := ""
	if slb != nil {
		s := slb.GetServer(parentAccountId)
		svrName = s.Name
	}
	// get sub_account
	var sanb witness.SubAccountNewBuilder
	subAccountMap, err := sanb.SubAccountNewMapFromTx(req.Tx)
	if err != nil {
		return nil, nil, fmt.Errorf("SubAccountNewMapFromTx err: %s", err.Error())
	}
	taskInfo := tables.TableTaskInfo{
		Id:              0,
		SvrName:         svrName,
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
		Retry:           0,
		CustomScripHash: "",
	}
	taskInfo.InitTaskId()

	var smtRecordList []tables.TableSmtRecordInfo
	for _, v := range subAccountMap {
		smtRecord := tables.TableSmtRecordInfo{
			Id:              0,
			SvrName:         svrName,
			AccountId:       v.SubAccountData.AccountId,
			Nonce:           v.CurrentSubAccountData.Nonce,
			RecordType:      tables.RecordTypeChain,
			RecordBN:        req.BlockNumber,
			TaskId:          taskInfo.TaskId,
			Action:          req.Action,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			Content:         "",
			RegisterYears:   0,
			RegisterArgs:    "",
			EditKey:         v.EditKey,
			Signature:       common.Bytes2Hex(v.Signature),
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       req.BlockTimestamp,
			SubAction:       v.Action,
			MintSignId:      "",
			ExpiredAt:       v.SignExpiredAt,
		}
		switch v.Action {
		case common.SubActionCreate:
			smtRecord.RegisterArgs = common.Bytes2Hex(v.SubAccountData.Lock.Args)
			smtRecord.RegisterYears = (v.SubAccountData.ExpiredAt - v.SubAccountData.RegisteredAt) / uint64(common.OneYearSec)
		case common.SubActionRenew:
			smtRecord.RenewYears = (v.CurrentSubAccountData.ExpiredAt - v.SubAccountData.ExpiredAt) / uint64(common.OneYearSec)
		case common.SubActionEdit:
			if len(v.EditLockArgs) > 0 {
				smtRecord.EditArgs = common.Bytes2Hex(v.EditLockArgs)
			}
			if len(v.EditRecords) > 0 {
				recordsBys, err := json.Marshal(v.EditRecords)
				if err != nil {
					return nil, nil, fmt.Errorf("records json.Marshal err: %s", err.Error())
				}
				smtRecord.EditRecords = string(recordsBys)
			}
		case common.SubActionRecycle:

		default:
			return nil, nil, fmt.Errorf("unknow sub-action [%s]", v.Action)
		}
		smtRecordList = append(smtRecordList, smtRecord)
	}
	return &taskInfo, smtRecordList, nil
}

func doNotifyDiscord(smtRecordList []tables.TableSmtRecordInfo) {
	content := ""
	count := 0
	var contentList []string
	for _, v := range smtRecordList {
		if v.SubAction != common.SubActionCreate {
			continue
		}
		account := v.Account
		registerYears := v.RegisterYears

		content += fmt.Sprintf(`** %s ** registered for %d year(s)
`, account, registerYears)
		count++
		if count == 30 {
			contentList = append(contentList, content)
			content = ""
			count = 0
		}
	}
	if content != "" {
		contentList = append(contentList, content)
	}
	go func() {
		for _, v := range contentList {
			if err := notify.SendNotifyDiscord(config.Cfg.Notify.DiscordCreateSubAccountKey, v); err != nil {
				log.Error("notify.SendNotifyDiscord err: ", err.Error(), v)
			}
		}
	}()
}

func (b *BlockParser) doNotifyLark(smtRecordList []tables.TableSmtRecordInfo) {
	content := ""
	count := 0
	var contentList []string
	for _, v := range smtRecordList {
		if v.SubAction != common.SubActionCreate {
			continue
		}
		account := v.Account
		registerYears := v.RegisterYears

		ownerNormal, _, _ := b.DasCore.Daf().ArgsToNormal(common.Hex2Bytes(v.RegisterArgs))
		owner := ownerNormal.AddressNormal
		if len(owner) > 4 {
			owner = owner[len(owner)-4:]
		}

		content += fmt.Sprintf(`%s, %d, %s
`, account, registerYears, owner)
		count++
		if count == 30 {
			contentList = append(contentList, content)
			content = ""
			count = 0
		}
	}
	if content != "" {
		contentList = append(contentList, content)
	}
	go func() {
		for _, v := range contentList {
			notify.SendLarkTextNotifyWithSvr(config.Cfg.Notify.LarkCreateSubAccountKey, "", v, false)
		}
	}()
}
