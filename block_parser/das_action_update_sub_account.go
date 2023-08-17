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
	"gorm.io/gorm"
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
	refOutpoint, outpoint, err := b.getOutpoint(req, common.DASContractNameSubAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("getOutpoint err: %s", err.Error())
		return
	}
	parentAccountId := ""
	for _, v := range req.Tx.Outputs {
		if v.Type != nil && contractSub.IsSameTypeId(v.Type.CodeHash) {
			parentAccountId = common.Bytes2Hex(v.Type.Args)
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

	approvalAction := make([]tables.TableSmtRecordInfo, 0)
	for _, v := range smtRecordList {
		switch v.SubAction {
		case common.SubActionCreateApproval, common.SubActionDelayApproval,
			common.SubActionRevokeApproval, common.SubActionFullfillApproval:
			approvalAction = append(approvalAction, v)
		}
	}

	// add task and smt records
	if selfTask.TaskId != "" {
		if err := b.DbDao.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(tables.TableTaskInfo{}).Where("task_id=?", selfTask.TaskId).
				Updates(map[string]interface{}{
					"task_type":    tables.TaskTypeChain,
					"smt_status":   tables.SmtStatusNeedToWrite,
					"tx_status":    tables.TxStatusCommitted,
					"block_number": req.BlockNumber,
				}).Error; err != nil {
				return err
			}
			if err := tx.Model(tables.TableSmtRecordInfo{}).
				Where("task_id=?", selfTask.TaskId).
				Updates(map[string]interface{}{
					"record_type": tables.RecordTypeChain,
					"RecordBN":    req.BlockNumber,
				}).Error; err != nil {
				return err
			}
			if err := b.doApprovalAction(req, tx, approvalAction); err != nil {
				return err
			}
			return nil
		}); err != nil {
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

func (b *BlockParser) doApprovalAction(req FuncTransactionHandleReq, tx *gorm.DB, smtRecordList []tables.TableSmtRecordInfo) error {
	refOutpoint, outpoint, err := b.getOutpoint(req, common.DASContractNameSubAccountCellType)
	if err != nil {
		return err
	}
	var sanb witness.SubAccountNewBuilder
	builderMap, err := sanb.SubAccountNewMapFromTx(req.Tx)
	if err != nil {
		return err
	}
	for _, v := range smtRecordList {
		accountInfo, err := b.DbDao.GetAccountInfoByAccountId(v.AccountId)
		if err != nil {
			return err
		}
		subAccBuilder, ok := builderMap[v.AccountId]
		if !ok {
			return fmt.Errorf("subAccBuilder not found: %s", v.AccountId)
		}
		subAccData := subAccBuilder.CurrentSubAccountData

		approval := tables.ApprovalInfo{
			BlockNumber: req.BlockNumber,
			RefOutpoint: refOutpoint,
			Outpoint:    outpoint,
		}
		switch v.SubAction {
		case common.SubActionCreateApproval:
			transfer := subAccData.AccountApproval.Params.Transfer
			dasf := core.DasAddressFormat{DasNetType: config.Cfg.Server.Net}
			toHex, _, err := dasf.ScriptToHex(transfer.ToLock)
			if err != nil {
				return err
			}
			platformHex, _, err := dasf.ScriptToHex(transfer.PlatformLock)
			if err != nil {
				return err
			}
			approval = tables.ApprovalInfo{
				BlockNumber:      req.BlockNumber,
				RefOutpoint:      refOutpoint,
				Outpoint:         outpoint,
				Account:          subAccData.Account(),
				AccountID:        subAccData.AccountId,
				Platform:         platformHex.AddressHex,
				OwnerAlgorithmID: accountInfo.OwnerAlgorithmId,
				Owner:            accountInfo.Owner,
				ToAlgorithmID:    toHex.DasAlgorithmId,
				To:               toHex.AddressHex,
				ProtectedUntil:   transfer.ProtectedUntil,
				SealedUntil:      transfer.SealedUntil,
				MaxDelayCount:    transfer.DelayCountRemain,
				Status:           tables.ApprovalStatusEnable,
			}
		case common.SubActionDelayApproval:
			approval, err = b.DbDao.GetAccountPendingApproval(subAccData.AccountId)
			if err != nil {
				return err
			}
			if approval.ID == 0 {
				return fmt.Errorf("approval not found")
			}
			transfer := subAccData.AccountApproval.Params.Transfer
			approval.SealedUntil = transfer.SealedUntil
			approval.PostponedCount++
		case common.SubActionRevokeApproval:
			approval, err = b.DbDao.GetAccountPendingApproval(subAccData.AccountId)
			if err != nil {
				return fmt.Errorf("GetAccountApprovalByOutpoint err: %s", err.Error())
			}
			if approval.ID == 0 {
				return fmt.Errorf("approval not found")
			}
			approval.Status = tables.ApprovalStatusRevoke
		case common.SubActionFullfillApproval:
			approval, err = b.DbDao.GetAccountPendingApproval(subAccData.AccountId)
			if err != nil {
				return fmt.Errorf("GetAccountApprovalByOutpoint err: %s", err.Error())
			}
			if approval.ID == 0 {
				return fmt.Errorf("approval not found")
			}
			approval.Status = tables.ApprovalStatusFulFill
		default:
			return fmt.Errorf("unknown sub action: %s", v.SubAction)
		}

		if err := tx.Save(&approval).Error; err != nil {
			return err
		}
	}
	return nil
}

func (b *BlockParser) getOutpoint(req FuncTransactionHandleReq, dasContractName common.DasContractName) (string, string, error) {
	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(dasContractName)
	if err != nil {
		return "", "", err
	}
	outpoint, refOutpoint := "", ""
	for i, v := range req.Tx.Outputs {
		if v.Type != nil && contractSub.IsSameTypeId(v.Type.CodeHash) {
			outpoint = common.OutPoint2String(req.TxHash, uint(i))
		}
	}
	for _, v := range req.Tx.Inputs {
		res, err := b.DasCore.Client().GetTransaction(b.Ctx, v.PreviousOutput.TxHash)
		if err != nil {
			return "", "", err
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}
	return refOutpoint, outpoint, nil
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
		case common.SubActionCreateApproval, common.SubActionDelayApproval,
			common.SubActionRevokeApproval, common.SubActionFullfillApproval:
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
