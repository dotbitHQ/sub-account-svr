package block_parser

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
)

func (b *BlockParser) DasActionCreateApproval(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionCreateApproval:", req.BlockNumber, req.TxHash)

	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
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
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	outpointStruct := common.String2OutPointStruct(outpoint)
	res, err := b.DasCore.Client().GetTransaction(b.Ctx, outpointStruct.TxHash)
	if err != nil {
		resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
		return
	}
	accountCellBuilder, err := witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderMapFromTx err: %s", err.Error())
		return
	}
	accountInfo, err := b.DbDao.GetAccountInfoByAccountId(accountCellBuilder.AccountId)
	if err != nil {
		resp.Err = fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
		return
	}

	transfer := accountCellBuilder.AccountApproval.Params.Transfer
	dasf := core.DasAddressFormat{DasNetType: config.Cfg.Server.Net}
	toHex, _, err := dasf.ScriptToHex(transfer.ToLock)
	if err != nil {
		resp.Err = fmt.Errorf("ScriptToHex err: %s", err.Error())
		return
	}
	platformHex, _, err := dasf.ScriptToHex(transfer.PlatformLock)
	if err != nil {
		resp.Err = fmt.Errorf("ScriptToHex err: %s", err.Error())
		return
	}

	if err := b.DbDao.CreateAccountApproval(tables.ApprovalInfo{
		BlockNumber:      req.BlockNumber,
		RefOutpoint:      refOutpoint,
		Outpoint:         outpoint,
		Account:          accountCellBuilder.Account,
		AccountID:        accountCellBuilder.AccountId,
		Action:           common.DasActionCreateApproval,
		Platform:         platformHex.AddressHex,
		OwnerAlgorithmID: accountInfo.OwnerAlgorithmId,
		Owner:            accountInfo.Owner,
		ToAlgorithmID:    toHex.DasAlgorithmId,
		To:               toHex.AddressHex,
		ProtectedUntil:   transfer.ProtectedUntil,
		SealedUntil:      transfer.SealedUntil,
		MaxDelayCount:    transfer.DelayCountRemain,
		Status:           tables.ApprovalStatusEnable,
	}); err != nil {
		resp.Err = fmt.Errorf("CreateAccountApproval err: %s", err.Error())
		return
	}
	return
}

func (b *BlockParser) DasActionDelayApproval(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionDelayApproval:", req.BlockNumber, req.TxHash)

	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
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
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	outpointStruct := common.String2OutPointStruct(outpoint)
	res, err := b.DasCore.Client().GetTransaction(b.Ctx, outpointStruct.TxHash)
	if err != nil {
		resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
		return
	}
	accountCellBuilder, err := witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderMapFromTx err: %s", err.Error())
		return
	}
	transfer := accountCellBuilder.AccountApproval.Params.Transfer

	approval, err := b.DbDao.GetAccountApprovalByOutpoint(refOutpoint)
	if err != nil {
		resp.Err = fmt.Errorf("GetAccountApprovalByOutpoint err: %s", err.Error())
		return
	}
	if approval.ID == 0 {
		resp.Err = fmt.Errorf("approval not found")
		return
	}
	approval.SealedUntil = transfer.SealedUntil
	approval.PostponedCount++

	resp.Err = b.DbDao.UpdateAccountApproval(approval.ID, map[string]interface{}{
		"outpoint":        outpoint,
		"ref_outpoint":    refOutpoint,
		"sealed_until":    approval.SealedUntil,
		"postponed_count": approval.PostponedCount,
	})
	return
}

func (b *BlockParser) DasActionRevokeApproval(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionApproval:", req.BlockNumber, req.TxHash)

	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
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
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	approval, err := b.DbDao.GetAccountApprovalByOutpoint(refOutpoint)
	if err != nil {
		resp.Err = fmt.Errorf("GetAccountApprovalByOutpoint err: %s", err.Error())
		return
	}
	if approval.ID == 0 {
		resp.Err = fmt.Errorf("approval not found")
		return
	}

	resp.Err = b.DbDao.UpdateAccountApproval(approval.ID, map[string]interface{}{
		"outpoint":     outpoint,
		"ref_outpoint": refOutpoint,
		"status":       tables.ApprovalStatusRevoke,
	})
	return
}

func (b *BlockParser) DasActionFulfillApproval(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionApproval:", req.BlockNumber, req.TxHash)

	// get ref outpoint
	contractSub, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
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
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	approval, err := b.DbDao.GetAccountApprovalByOutpoint(refOutpoint)
	if err != nil {
		resp.Err = fmt.Errorf("GetAccountApprovalByOutpoint err: %s", err.Error())
		return
	}
	if approval.ID == 0 {
		resp.Err = fmt.Errorf("approval not found")
		return
	}

	resp.Err = b.DbDao.UpdateAccountApproval(approval.ID, map[string]interface{}{
		"outpoint":     outpoint,
		"ref_outpoint": refOutpoint,
		"status":       tables.ApprovalStatusFulFill,
	})
	return
}
