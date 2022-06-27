package block_parser

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
)

func (b *BlockParser) DasActionEnableSubAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version enable sub account tx")
		return
	}
	log.Info("DasActionEnableSubAccount:", req.BlockNumber, req.TxHash)

	accBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}

	task := tables.TableTaskInfo{
		Id:              0,
		TaskId:          "",
		TaskType:        tables.TaskTypeChain,
		ParentAccountId: accBuilder.AccountId,
		Action:          common.DasActionEnableSubAccount,
		RefOutpoint:     "",
		BlockNumber:     req.BlockNumber,
		Outpoint:        common.OutPoint2String(req.TxHash, 1),
		Timestamp:       req.BlockTimestamp,
		SmtStatus:       tables.SmtStatusWriteComplete,
		TxStatus:        tables.TxStatusCommitted,
	}
	task.InitTaskId()

	if err := b.DbDao.CreateTaskByDasActionEnableSubAccount(&task); err != nil {
		resp.Err = fmt.Errorf("CreateTaskByDasActionEnableSubAccount err: %s", err.Error())
		return
	}
	return
}
