package block_parser

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"gorm.io/gorm"
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

func (b *BlockParser) DasActionConfigSubAccountOrCustomScript(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	isCV, index, err := CurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version enable sub account tx")
		return
	}
	parentAccountId := common.Bytes2Hex(req.Tx.Outputs[index].Type.Args)

	log.Info("DasActionConfigSubAccountOrCustomScript:", req.Action, req.BlockNumber, req.TxHash, parentAccountId)

	accBuilder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}

	if err := b.DbDao.Transaction(func(tx *gorm.DB) error {
		task := &tables.TableTaskInfo{
			TaskType:        tables.TaskTypeChain,
			ParentAccountId: accBuilder.AccountId,
			Action:          req.Action,
			BlockNumber:     req.BlockNumber,
			Outpoint:        common.OutPoint2String(req.TxHash, 1),
			Timestamp:       req.BlockTimestamp,
			SmtStatus:       tables.SmtStatusWriteComplete,
			TxStatus:        tables.TxStatusCommitted,
		}
		task.InitTaskId()
		if err := tx.Where("ref_outpoint='' AND outpoint=?", task.Outpoint).
			Delete(&tables.TableTaskInfo{}).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}

		if err := tx.Model(&tables.RuleWhitelist{}).
			Where("tx_hash=? and tx_status=?", req.TxHash, tables.TxStatusPending).
			Updates(map[string]interface{}{
				"tx_status":       tables.TxStatusCommitted,
				"block_number":    req.BlockNumber,
				"block_timestamp": req.BlockTimestamp,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&tables.RuleWhitelist{}).
			Where("parent_account_id=? and tx_status=?", parentAccountId, tables.TxStatusPending).
			Delete(&tables.RuleWhitelist{}).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		return nil
	}); err != nil {
		resp.Err = err
		return
	}
	return
}

func (b *BlockParser) DasActionCollectSubAccountProfit(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version enable sub account tx")
		return
	}
	log.Info("DasActionCollectSubAccountProfit:", req.BlockNumber, req.TxHash)

	task := tables.TableTaskInfo{
		Id:              0,
		TaskId:          "",
		TaskType:        tables.TaskTypeChain,
		ParentAccountId: common.Bytes2Hex(req.Tx.Outputs[0].Type.Args),
		Action:          common.DasActionCollectSubAccountProfit,
		RefOutpoint:     "",
		BlockNumber:     req.BlockNumber,
		Outpoint:        common.OutPoint2String(req.TxHash, 0),
		Timestamp:       req.BlockTimestamp,
		SmtStatus:       tables.SmtStatusWriteComplete,
		TxStatus:        tables.TxStatusCommitted,
	}
	task.InitTaskId()

	if err := b.DbDao.CreateTaskByProfitWithdraw(&task); err != nil {
		resp.Err = fmt.Errorf("CreateTaskByProfitWithdraw err: %s", err.Error())
		return
	}
	return
}
